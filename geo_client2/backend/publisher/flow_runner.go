package publisher

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"geo_client2/backend/logger"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/proto"
)

type FlowRunner struct {
	log      *logger.Logger
	platform string
	emit     EventEmitter
	resume   <-chan struct{}
	tempVars map[string]string
}

func NewFlowRunner(log *logger.Logger, platform string, emit EventEmitter, resume <-chan struct{}) *FlowRunner {
	return &FlowRunner{
		log:      log,
		platform: platform,
		emit:     emit,
		resume:   resume,
		tempVars: make(map[string]string),
	}
}

func (r *FlowRunner) Run(ctx context.Context, page *rod.Page, flow *Flow, article Article) (string, error) {
	if page == nil {
		return "", fmt.Errorf("nil page")
	}
	if flow == nil {
		return "", fmt.Errorf("nil flow")
	}

	r.log.InfoWithContext(ctx, fmt.Sprintf("[Flow] starting platform=%s steps=%d", r.platform, len(flow.Steps)), map[string]interface{}{
		"platform": r.platform,
		"steps":    len(flow.Steps),
	}, nil)

	for i, step := range flow.Steps {
		stepErr := r.runStep(ctx, page, step, article)
		if stepErr != nil {
			if step.Optional {
				r.log.InfoWithContext(ctx, fmt.Sprintf("[Flow] optional step skipped id=%s action=%s err=%v", step.ID, step.Action, stepErr), map[string]interface{}{
					"platform": r.platform,
					"step_id":  step.ID,
					"action":   step.Action,
					"index":    i,
					"optional": true,
					"error":    stepErr.Error(),
				}, nil)
				continue
			}
			r.log.InfoWithContext(ctx, fmt.Sprintf("[Flow] step failed id=%s action=%s err=%v", step.ID, step.Action, stepErr), map[string]interface{}{
				"platform": r.platform,
				"step_id":  step.ID,
				"action":   step.Action,
				"index":    i,
				"error":    stepErr.Error(),
			}, nil)
			return "", fmt.Errorf("flow step failed id=%s action=%s: %w", step.ID, step.Action, stepErr)
		}
	}

	info := page.MustInfo()
	if info == nil {
		return "", nil
	}
	r.log.InfoWithContext(ctx, fmt.Sprintf("[Flow] completed platform=%s url=%s", r.platform, info.URL), map[string]interface{}{
		"platform":    r.platform,
		"article_url": info.URL,
	}, nil)
	return info.URL, nil
}

func (r *FlowRunner) runStep(ctx context.Context, page *rod.Page, step FlowStep, article Article) error {
	stepCtx := ctx
	var cancel func()
	if step.TimeoutMS > 0 {
		stepCtx, cancel = context.WithTimeout(ctx, time.Duration(step.TimeoutMS)*time.Millisecond)
		defer cancel()
	}

	action := strings.ToLower(strings.TrimSpace(step.Action))

	startedAt := time.Now()
	stepDetails := map[string]interface{}{
		"platform": r.platform,
		"step_id":  step.ID,
		"action":   action,
	}
	if step.URL != "" {
		stepDetails["url"] = step.URL
	}
	if step.Selector != "" {
		stepDetails["selector"] = step.Selector
	}
	if step.Regex != "" {
		stepDetails["regex"] = step.Regex
	}
	if step.MS > 0 {
		stepDetails["ms"] = step.MS
	}
	r.log.InfoWithContext(ctx, fmt.Sprintf("[Flow] step start id=%s action=%s platform=%s", step.ID, action, r.platform), stepDetails, nil)

	err := r.doStep(stepCtx, ctx, page, step, action, article)

	elapsed := int(time.Since(startedAt).Milliseconds())
	if err != nil {
		r.log.InfoWithContext(ctx, fmt.Sprintf("[Flow] step error id=%s action=%s elapsed=%dms err=%v", step.ID, action, elapsed, err), map[string]interface{}{
			"platform":   r.platform,
			"step_id":    step.ID,
			"action":     action,
			"elapsed_ms": elapsed,
			"error":      err.Error(),
		}, nil)
	} else {
		r.log.InfoWithContext(ctx, fmt.Sprintf("[Flow] step ok id=%s action=%s elapsed=%dms", step.ID, action, elapsed), map[string]interface{}{
			"platform":   r.platform,
			"step_id":    step.ID,
			"action":     action,
			"elapsed_ms": elapsed,
		}, nil)
	}
	return err
}

func (r *FlowRunner) interp(raw string, article Article) string {
	return interpolateValue(raw, article, r.tempVars)
}

func (r *FlowRunner) doStep(stepCtx context.Context, ctx context.Context, page *rod.Page, step FlowStep, action string, article Article) error {
	switch action {
	case "navigate":
		if step.URL == "" {
			return fmt.Errorf("missing url")
		}
		return page.Navigate(step.URL)
	case "wait_load":
		page.MustWaitLoad()
		return nil
	case "wait_idle":
		page.MustWaitIdle()
		return nil
	case "wait":
		d := time.Duration(step.MS) * time.Millisecond
		if d <= 0 {
			d = 1000 * time.Millisecond
		}
		select {
		case <-stepCtx.Done():
			return stepCtx.Err()
		case <-time.After(d):
			return nil
		}
	case "wait_selector":
		if step.Selector == "" {
			return fmt.Errorf("missing selector")
		}
		to := 10 * time.Second
		if step.TimeoutMS > 0 {
			to = time.Duration(step.TimeoutMS) * time.Millisecond
		}
		_, err := page.Timeout(to).Element(step.Selector)
		return err
	case "click":
		el, err := r.findElement(page, step)
		if err != nil {
			return err
		}
		return el.Click(proto.InputMouseButtonLeft, 1)
	case "click_r":
		if step.Selector == "" || step.Regex == "" {
			return fmt.Errorf("missing selector or regex")
		}
		re, err := regexp.Compile(step.Regex)
		if err != nil {
			return fmt.Errorf("compile regex: %w", err)
		}
		_ = re
		el, err := page.ElementR(step.Selector, step.Regex)
		if err != nil {
			return err
		}
		return el.Click(proto.InputMouseButtonLeft, 1)
	case "fill":
		el, err := r.findElement(page, step)
		if err != nil {
			return err
		}
		if step.ClickFirst {
			_ = el.Click(proto.InputMouseButtonLeft, 1)
		}
		val := r.interp(step.Value, article)
		return r.fillElement(el, step.Mode, val)
	case "frame_fill":
		if step.Frame == "" {
			return fmt.Errorf("missing frameSelector")
		}
		if step.Selector == "" {
			return fmt.Errorf("missing selector")
		}
		iframeEl, err := page.Element(step.Frame)
		if err != nil {
			return fmt.Errorf("find iframe: %w", err)
		}
		frame, err := iframeEl.Frame()
		if err != nil {
			return fmt.Errorf("get iframe frame: %w", err)
		}
		el, err := frame.Element(step.Selector)
		if err != nil {
			return fmt.Errorf("find frame element: %w", err)
		}
		if step.ClickFirst {
			_ = el.Click(proto.InputMouseButtonLeft, 1)
		}
		val := r.interp(step.Value, article)
		return r.fillElement(el, step.Mode, val)
	case "eval":
		if strings.TrimSpace(step.Script) == "" {
			return fmt.Errorf("missing script")
		}
		script := r.interp(step.Script, article)
		args := make([]interface{}, 0, len(step.Args))
		for _, a := range step.Args {
			args = append(args, r.interp(a, article))
		}
		_, err := page.Eval(script, args...)
		return err
	case "set_files":
		if step.Selector == "" {
			return fmt.Errorf("missing selector")
		}
		if len(step.Files) == 0 {
			return fmt.Errorf("missing files")
		}
		files := make([]string, 0, len(step.Files))
		for _, f := range step.Files {
			files = append(files, r.interp(f, article))
		}
		el, err := page.Element(step.Selector)
		if err != nil {
			return err
		}
		return el.SetFiles(files)
	case "wait_url_not":
		if step.URL == "" {
			return fmt.Errorf("missing url")
		}
		deadline := time.Now().Add(10 * time.Second)
		if step.TimeoutMS > 0 {
			deadline = time.Now().Add(time.Duration(step.TimeoutMS) * time.Millisecond)
		}
		for time.Now().Before(deadline) {
			select {
			case <-stepCtx.Done():
				return stepCtx.Err()
			case <-time.After(500 * time.Millisecond):
			}
			info := page.MustInfo()
			if info != nil && info.URL != "" && info.URL != step.URL {
				return nil
			}
		}
		return fmt.Errorf("url did not change")
	case "wait_url_change":
		deadline := time.Now().Add(15 * time.Second)
		if step.TimeoutMS > 0 {
			deadline = time.Now().Add(time.Duration(step.TimeoutMS) * time.Millisecond)
		}
		startURL := ""
		if info := page.MustInfo(); info != nil {
			startURL = info.URL
		}
		for time.Now().Before(deadline) {
			select {
			case <-stepCtx.Done():
				return stepCtx.Err()
			case <-time.After(500 * time.Millisecond):
			}
			info := page.MustInfo()
			if info != nil && info.URL != "" && info.URL != startURL {
				return nil
			}
		}
		return fmt.Errorf("url did not change from %s", startURL)
	case "download_to_temp":
		urlStr := r.interp(step.URL, article)
		if urlStr == "" {
			return fmt.Errorf("download_to_temp: missing url")
		}
		path, err := downloadURLToTempFileFlow(urlStr)
		if err != nil {
			return fmt.Errorf("download_to_temp: %w", err)
		}
		r.tempVars["temp_cover"] = path
		r.log.InfoWithContext(ctx, fmt.Sprintf("[Flow] download_to_temp ok path=%s", path), map[string]interface{}{
			"platform": r.platform,
			"step_id":  step.ID,
			"path":     path,
		}, nil)
		return nil
	case "needs_manual":
		if r.emit == nil || r.resume == nil {
			return fmt.Errorf("needs_manual: no emit/resume wired")
		}
		prompt := step.Prompt
		if prompt == "" {
			prompt = "请手动完成操作后点击继续"
		}
		taskID := fmt.Sprintf("%s-%d", r.platform, time.Now().UnixNano())
		return emitNeedsManual(ctx, r.platform, taskID, prompt, r.resume, r.emit)
	default:
		return fmt.Errorf("unsupported action: %s", step.Action)
	}
}

func (r *FlowRunner) findElement(page *rod.Page, step FlowStep) (*rod.Element, error) {
	if step.Selector == "" {
		return nil, fmt.Errorf("missing selector")
	}
	return page.Element(step.Selector)
}

func (r *FlowRunner) fillElement(el *rod.Element, mode string, value string) error {
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
		_, err := el.Eval(`(el, v) => { el.value = v; el.dispatchEvent(new Event('input', { bubbles: true })); }`, value)
		return err
	case "value":
		_, err := el.Eval(`(el, v) => { el.value = v; el.dispatchEvent(new Event('input', { bubbles: true })); el.dispatchEvent(new Event('change', { bubbles: true })); }`, value)
		return err
	case "innertext":
		_, err := el.Eval(`(el, v) => { el.innerText = v; el.dispatchEvent(new Event('input', { bubbles: true })); el.dispatchEvent(new Event('change', { bubbles: true })); }`, value)
		return err
	case "innerhtml":
		_, err := el.Eval(`(el, v) => { el.innerHTML = v; el.dispatchEvent(new Event('input', { bubbles: true })); el.dispatchEvent(new Event('change', { bubbles: true })); }`, value)
		return err
	default:
		return fmt.Errorf("unsupported fill mode: %s", mode)
	}
}

func downloadURLToTempFileFlow(urlStr string) (string, error) {
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
	f, err := os.CreateTemp("", "geo_flow_cover_*"+ext)
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
