package scrape

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"geo_client2/backend/logger"
	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/proto"
)

type Runner struct {
	log      *logger.Logger
	platform string
	data     *FlowResult
}

func NewRunner(log *logger.Logger, platform string) *Runner {
	return &Runner{
		log:      log,
		platform: platform,
		data:     &FlowResult{},
	}
}

func (r *Runner) Result() *FlowResult {
	return r.data
}

func (r *Runner) Run(ctx context.Context, page *rod.Page, flow *ScrapeFlow, vars map[string]string) error {
	if page == nil {
		return fmt.Errorf("nil page")
	}
	if flow == nil {
		return fmt.Errorf("nil flow")
	}
	if vars == nil {
		vars = map[string]string{}
	}
	for i, raw := range flow.Pipeline {
		if err := r.runStep(ctx, page, flow, i, raw, vars); err != nil {
			return err
		}
	}
	return nil
}

func (r *Runner) runStep(ctx context.Context, page *rod.Page, flow *ScrapeFlow, index int, raw map[string]interface{}, vars map[string]string) error {
	step := mapString(raw)
	action := strings.ToLower(strings.TrimSpace(step["action"]))
	if action == "" {
		return fmt.Errorf("flow step[%d] missing action", index)
	}

	start := time.Now()
	err := r.doStep(ctx, page, action, step, vars)
	if err != nil {
		r.log.InfoWithContext(ctx, "[ScrapeFlow] step error", map[string]interface{}{
			"platform": r.platform,
			"index":    index,
			"action":   action,
			"error":    err.Error(),
		}, nil)
		return err
	}
	_ = start
	return nil
}

func (r *Runner) doStep(ctx context.Context, page *rod.Page, action string, step map[string]string, vars map[string]string) error {
	switch action {
	case "navigate":
		url := interp(step["url"], vars)
		if url == "" {
			return fmt.Errorf("navigate: missing url")
		}
		return page.Navigate(url)
	case "wait_load":
		return page.WaitLoad()
	case "wait_idle":
		rod.Try(func() { _ = page.Timeout(10 * time.Second).WaitStable(1 * time.Second) })
		return nil
	case "wait_ms":
		ms := parseInt(step["ms"], 1000)
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(time.Duration(ms) * time.Millisecond):
			return nil
		}
	case "wait":
		ms := parseInt(step["ms"], 1000)
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(time.Duration(ms) * time.Millisecond):
			return nil
		}
	case "wait_selector":
		selector := step["selector"]
		if selector == "" {
			return fmt.Errorf("wait_selector: missing selector")
		}
		to := time.Duration(parseInt(step["timeoutMs"], 10000)) * time.Millisecond
		_, err := page.Timeout(to).Element(selector)
		return err
	case "click":
		selector := step["selector"]
		if selector == "" {
			return fmt.Errorf("click: missing selector")
		}
		el, err := page.Element(selector)
		if err != nil {
			return err
		}
		return el.Click(proto.InputMouseButtonLeft, 1)
	case "click_r":
		selector := step["selector"]
		regex := step["regex"]
		if selector == "" || regex == "" {
			return fmt.Errorf("click_r: missing selector or regex")
		}
		re, err := regexp.Compile(regex)
		if err != nil {
			return fmt.Errorf("click_r: compile regex: %w", err)
		}
		_ = re
		el, err := page.ElementR(selector, regex)
		if err != nil {
			return err
		}
		return el.Click(proto.InputMouseButtonLeft, 1)
	case "fill":
		selector := step["selector"]
		value := interp(step["value"], vars)
		if selector == "" {
			return fmt.Errorf("fill: missing selector")
		}
		el, err := page.Element(selector)
		if err != nil {
			return err
		}
		return fillElement(el, step["mode"], value)
	case "frame_fill":
		frameSel := step["frameSelector"]
		selector := step["selector"]
		value := interp(step["value"], vars)
		if frameSel == "" || selector == "" {
			return fmt.Errorf("frame_fill: missing frameSelector or selector")
		}
		iframe, err := page.Element(frameSel)
		if err != nil {
			return fmt.Errorf("frame_fill: find iframe: %w", err)
		}
		frame, err := iframe.Frame()
		if err != nil {
			return fmt.Errorf("frame_fill: get frame: %w", err)
		}
		el, err := frame.Element(selector)
		if err != nil {
			return fmt.Errorf("frame_fill: find element: %w", err)
		}
		return fillElement(el, step["mode"], value)
	case "eval":
		script := interp(step["script"], vars)
		if strings.TrimSpace(script) == "" {
			return fmt.Errorf("eval: missing script")
		}
		_, err := page.Eval(script)
		return err
	case "set_files":
		selector := step["selector"]
		filesRaw := step["files"]
		if selector == "" || strings.TrimSpace(filesRaw) == "" {
			return fmt.Errorf("set_files: missing selector or files")
		}
		files := splitCSV(interp(filesRaw, vars))
		el, err := page.Element(selector)
		if err != nil {
			return err
		}
		return el.SetFiles(files)
	case "wait_url_not":
		urlStr := step["url"]
		if urlStr == "" {
			return fmt.Errorf("wait_url_not: missing url")
		}
		deadline := time.Now().Add(time.Duration(parseInt(step["timeoutMs"], 10000)) * time.Millisecond)
		for time.Now().Before(deadline) {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(500 * time.Millisecond):
			}
			info := page.MustInfo()
			if info != nil && info.URL != "" && info.URL != urlStr {
				return nil
			}
		}
		return fmt.Errorf("wait_url_not: url did not change")
	case "wait_url_change":
		deadline := time.Now().Add(time.Duration(parseInt(step["timeoutMs"], 15000)) * time.Millisecond)
		startURL := ""
		if info := page.MustInfo(); info != nil {
			startURL = info.URL
		}
		for time.Now().Before(deadline) {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(500 * time.Millisecond):
			}
			info := page.MustInfo()
			if info != nil && info.URL != "" && info.URL != startURL {
				return nil
			}
		}
		return fmt.Errorf("wait_url_change: url did not change from %s", startURL)
	case "download_to_temp":
		urlStr := interp(step["url"], vars)
		if urlStr == "" {
			return fmt.Errorf("download_to_temp: missing url")
		}
		path, err := downloadURLToTemp(urlStr)
		if err != nil {
			return err
		}
		vars["temp_file"] = path
		return nil
	case "needs_manual":
		return fmt.Errorf("needs_manual: manual step not supported in scrape flow runner")
	case "dom_extract":
		return r.domExtract(page, step, vars)
	case "normalize":
		return r.normalize()
	case "parse_json":
		key := step["var"]
		value := step["value"]
		if key == "" || value == "" {
			return fmt.Errorf("parse_json: missing var or value")
		}
		vars[key] = value
		return nil
	case "set_result":
		return r.applyResult(step, vars)
	default:
		return fmt.Errorf("unsupported action: %s", action)
	}
}

func (r *Runner) applyResult(step map[string]string, vars map[string]string) error {
	queries := splitCSV(interp(step["queries"], vars))
	fullText := interp(step["full_text"], vars)

	if qRaw := strings.TrimSpace(step["queries_json"]); qRaw != "" {
		if err := json.Unmarshal([]byte(qRaw), &queries); err != nil {
			return fmt.Errorf("set_result: parse queries_json: %w", err)
		}
	}

	if citationsRaw := strings.TrimSpace(step["citations_json"]); citationsRaw != "" {
		var cites []FlowCitation
		if err := json.Unmarshal([]byte(citationsRaw), &cites); err != nil {
			return fmt.Errorf("set_result: parse citations_json: %w", err)
		}
		r.data.Citations = cites
	}

	if len(queries) > 0 {
		r.data.Queries = queries
	}
	if fullText != "" {
		r.data.FullText = fullText
	}
	return nil
}

func (r *Runner) domExtract(page *rod.Page, step map[string]string, vars map[string]string) error {
	selectors := splitCSV(step["selectors"])
	if len(selectors) == 0 {
		selectors = []string{"a[href]"}
	}
	limit := parseInt(step["limit"], 50)
	query := strings.TrimSpace(interp(step["query"], vars))
	queryIndex := 0
	if q := strings.TrimSpace(step["queryIndex"]); q != "" {
		queryIndex = parseInt(q, 0)
	}
	if query != "" && len(r.data.Queries) == 0 {
		r.data.Queries = []string{query}
	}

	seen := map[string]bool{}
	for _, sel := range selectors {
		els, err := page.Elements(sel)
		if err != nil {
			continue
		}
		for _, el := range els {
			if len(r.data.Citations) >= limit {
				break
			}
			href, _ := el.Attribute("href")
			if href == nil || strings.TrimSpace(*href) == "" {
				continue
			}
			urlStr := strings.TrimSpace(*href)
			if seen[urlStr] {
				continue
			}
			seen[urlStr] = true
			title, _ := el.Text()
			r.data.Citations = append(r.data.Citations, FlowCitation{
				URL:          urlStr,
				Title:        strings.TrimSpace(title),
				QueryIndexes: []int{queryIndex},
				Query:        query,
				CiteIndex:    len(r.data.Citations) + 1,
			})
		}
	}
	return nil
}

func (r *Runner) normalize() error {
	for i := range r.data.Citations {
		c := &r.data.Citations[i]
		if c.Domain == "" {
			if u, err := url.Parse(c.URL); err == nil {
				c.Domain = u.Hostname()
			}
		}
	}
	return nil
}

func mapString(raw map[string]interface{}) map[string]string {
	out := map[string]string{}
	for k, v := range raw {
		out[k] = fmt.Sprintf("%v", v)
	}
	return out
}

func interp(raw string, vars map[string]string) string {
	if raw == "" || vars == nil {
		return raw
	}
	out := raw
	for k, v := range vars {
		out = strings.ReplaceAll(out, "{{"+k+"}}", v)
	}
	return out
}

func parseInt(raw string, def int) int {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return def
	}
	var v int
	_, err := fmt.Sscanf(raw, "%d", &v)
	if err != nil {
		return def
	}
	return v
}

func splitCSV(raw string) []string {
	parts := []string{}
	for _, p := range strings.Split(raw, ",") {
		p = strings.TrimSpace(p)
		if p != "" {
			parts = append(parts, p)
		}
	}
	return parts
}

func fillElement(el *rod.Element, mode string, value string) error {
	m := strings.ToLower(strings.TrimSpace(mode))
	if m == "" {
		m = "input"
	}
	switch m {
	case "input":
		_ = el.Focus()
		if err := el.Input(value); err == nil {
			return nil
		}
		_, err := el.Eval(`(v) => { this.value = v; this.dispatchEvent(new Event('input', { bubbles: true })); }`, value)
		return err
	case "value":
		_, err := el.Eval(`(v) => { this.value = v; this.dispatchEvent(new Event('input', { bubbles: true })); this.dispatchEvent(new Event('change', { bubbles: true })); }`, value)
		return err
	case "innertext":
		_, err := el.Eval(`(v) => { this.innerText = v; this.dispatchEvent(new Event('input', { bubbles: true })); this.dispatchEvent(new Event('change', { bubbles: true })); }`, value)
		return err
	case "innerhtml":
		_, err := el.Eval(`(v) => { this.innerHTML = v; this.dispatchEvent(new Event('input', { bubbles: true })); this.dispatchEvent(new Event('change', { bubbles: true })); }`, value)
		return err
	case "clipboard":
		_ = el.Click(proto.InputMouseButtonLeft, 1)
		plain := stripHTMLTags(value)
		_, err := el.Eval(`(html, plain) => {
			this.focus();
			const sel = window.getSelection();
			const range = document.createRange();
			range.selectNodeContents(this);
			sel.removeAllRanges();
			sel.addRange(range);
			const dt = new DataTransfer();
			dt.setData('text/html', html);
			dt.setData('text/plain', plain);
			this.dispatchEvent(new ClipboardEvent('paste', { clipboardData: dt, bubbles: true, cancelable: true }));
		}`, value, plain)
		return err
	case "react":
		_, err := el.Eval(`(v) => {
			const nativeSetter = Object.getOwnPropertyDescriptor(window.HTMLTextAreaElement.prototype, 'value') ||
				Object.getOwnPropertyDescriptor(window.HTMLInputElement.prototype, 'value');
			if (nativeSetter && nativeSetter.set) {
				nativeSetter.set.call(this, v);
			} else {
				this.value = v;
			}
			this.dispatchEvent(new Event('input', { bubbles: true }));
			this.dispatchEvent(new Event('change', { bubbles: true }));
		}`, value)
		return err
	default:
		return fmt.Errorf("unsupported fill mode: %s", mode)
	}
}

var reHTMLTag = regexp.MustCompile(`<[^>]*>`)

func stripHTMLTags(html string) string {
	plain := reHTMLTag.ReplaceAllString(html, "")
	plain = strings.ReplaceAll(plain, "&amp;", "&")
	plain = strings.ReplaceAll(plain, "&lt;", "<")
	plain = strings.ReplaceAll(plain, "&gt;", ">")
	plain = strings.ReplaceAll(plain, "&nbsp;", " ")
	plain = strings.ReplaceAll(plain, "&quot;", "\"")
	plain = strings.ReplaceAll(plain, "&#39;", "'")
	return strings.TrimSpace(plain)
}

func downloadURLToTemp(urlStr string) (string, error) {
	if !strings.HasPrefix(urlStr, "http://") && !strings.HasPrefix(urlStr, "https://") {
		return "", fmt.Errorf("unsupported url scheme: %s", urlStr)
	}
	client := &http.Client{Timeout: 20 * time.Second}
	resp, err := client.Get(urlStr)
	if err != nil {
		return "", fmt.Errorf("http get: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("http status %d", resp.StatusCode)
	}
	ext := strings.ToLower(filepath.Ext(urlStr))
	if ext == "" || len(ext) > 6 {
		ext = ".png"
	}
	f, err := os.CreateTemp("", "geo_scrape_flow_*"+ext)
	if err != nil {
		return "", fmt.Errorf("create temp: %w", err)
	}
	defer f.Close()
	if _, err := io.Copy(f, resp.Body); err != nil {
		_ = os.Remove(f.Name())
		return "", fmt.Errorf("write temp: %w", err)
	}
	return f.Name(), nil
}
