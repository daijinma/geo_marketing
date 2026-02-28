package publisher

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"geo_client2/backend/aiassist"
	"geo_client2/backend/config"
	"geo_client2/backend/logger"
	"geo_client2/backend/provider"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/proto"
)

// BasePublisher provides common browser lifecycle for publishing.
type BasePublisher struct {
	platform  string
	accountID string
	factory   *provider.Factory
	prov      provider.Provider
	baseProv  *provider.BaseProvider
	logger    *logger.Logger
}

// NewPublisher creates a platform-specific publisher.
func NewPublisher(platform string, factory *provider.Factory, accountID string) (Publisher, error) {
	base := &BasePublisher{
		platform:  platform,
		accountID: accountID,
		factory:   factory,
		logger:    logger.GetLogger(),
	}

	// Prefer config-driven flow when available for this platform.
	if _, err := LoadPublishFlow(platform); err == nil {
		return &FlowPublisher{BasePublisher: base}, nil
	}

	switch platform {
	case "zhihu":
		return &ZhihuPublisher{BasePublisher: base}, nil
	case "sohu":
		return &SohuPublisher{BasePublisher: base}, nil
	case "csdn":
		return &CsdnPublisher{BasePublisher: base}, nil
	case "qie":
		return &QiePublisher{BasePublisher: base}, nil
	case "baijiahao":
		return &BaijiaPublisher{BasePublisher: base}, nil
	case "xiaohongshu":
		return &XhsPublisher{BasePublisher: base}, nil
	default:
		return nil, fmt.Errorf("unsupported publish platform: %s", platform)
	}
}

func (b *BasePublisher) getProvider() (provider.Provider, error) {
	if b.prov != nil {
		return b.prov, nil
	}
	prov, err := b.factory.GetProvider(b.platform, false, 60000, b.accountID)
	if err != nil {
		return nil, err
	}
	b.prov = prov
	return prov, nil
}

func (b *BasePublisher) getBaseProvider() *provider.BaseProvider {
	if b.baseProv != nil {
		return b.baseProv
	}
	headless := false
	if b.factory != nil {
		headless = b.factory.IsHeadless()
	}
	b.baseProv = provider.NewBaseProvider(b.platform, headless, 60000, b.accountID)
	return b.baseProv
}

func (b *BasePublisher) CheckLoginStatus() (bool, error) {
	prov, err := b.getProvider()
	if err != nil {
		return false, err
	}
	return prov.CheckLoginStatus()
}

func (b *BasePublisher) GetLoginUrl() string {
	prov, err := b.getProvider()
	if err != nil {
		return ""
	}
	return prov.GetLoginUrl()
}

func (b *BasePublisher) StartLogin() (func(), error) {
	prov, err := b.getProvider()
	if err != nil {
		return nil, err
	}
	return prov.StartLogin()
}

func (b *BasePublisher) Close() error {
	if b.prov != nil {
		return b.prov.Close()
	}
	return nil
}

// waitForResume blocks until the resume channel receives a signal or ctx is cancelled.
func waitForResume(ctx context.Context, resume <-chan struct{}) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-resume:
		return nil
	}
}

// emitNeedsManual emits the needs_manual event and waits for resume.
func emitNeedsManual(ctx context.Context, platform, taskID, prompt string, resume <-chan struct{}, emit EventEmitter) error {
	emit("publish:needs_manual", map[string]interface{}{
		"platform": platform,
		"prompt":   prompt,
		"taskId":   taskID,
	})
	return waitForResume(ctx, resume)
}

type Observation struct {
	Goal       string `json:"goal"`
	URL        string `json:"url"`
	Title      string `json:"title"`
	DOM        string `json:"dom"`
	Clickables string `json:"clickables"`
	Screenshot string `json:"screenshot"`
}

type AIPublishConfig struct {
	Enabled bool
	BaseURL string
	APIKey  string
}

func buildGoal(article Article) string {
	goal := fmt.Sprintf("发布文章。标题：%s。正文：%s", article.Title, article.Content)
	if article.CoverImage != "" {
		goal += fmt.Sprintf("。封面图：%s", article.CoverImage)
	}
	return goal
}

func limitString(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max]
}

func captureObservation(page *rod.Page, goal string) (Observation, error) {
	info := page.MustInfo()
	url := ""
	if info != nil {
		url = info.URL
	}

	titleStr := page.MustEval("() => document.title").String()
	textStr := page.MustEval("() => document.body ? document.body.innerText : ''").String()
	textStr = limitString(textStr, 4000)
	htmlStr := page.MustEval("() => document.body ? document.body.innerHTML : ''").String()
	htmlStr = limitString(htmlStr, 4000)

	clickablesJSON := page.MustEval(`() => {
  const isVisible = (el) => {
    const rect = el.getBoundingClientRect();
    return rect.width > 0 && rect.height > 0;
  };
  const cssPath = (el) => {
    if (!el || !el.tagName) return '';
    if (el.id) return '#' + el.id;
    const parts = [];
    let node = el;
    while (node && node.tagName && parts.length < 4) {
      let selector = node.tagName.toLowerCase();
      if (node.className && typeof node.className === 'string') {
        const cls = node.className.trim().split(/\s+/).slice(0, 2).join('.');
        if (cls) selector += '.' + cls;
      }
      if (node.parentElement) {
        const siblings = Array.from(node.parentElement.children).filter(n => n.tagName === node.tagName);
        if (siblings.length > 1) {
          const index = siblings.indexOf(node) + 1;
          selector += ':nth-of-type(' + index + ')';
        }
      }
      parts.unshift(selector);
      node = node.parentElement;
    }
    return parts.join(' > ');
  };
  const nodes = Array.from(document.querySelectorAll('button, a, [role="button"], input[type="submit"], input[type="button"]'));
  const list = [];
  for (const el of nodes) {
    if (!isVisible(el)) continue;
    const text = (el.innerText || el.value || '').trim();
    list.push({ text, selector: cssPath(el), tag: el.tagName.toLowerCase() });
    if (list.length >= 60) break;
  }
  return JSON.stringify(list);
}`).String()

	clickablesJSON = limitString(clickablesJSON, 4000)

	domStr := fmt.Sprintf("[TEXT]\n%s\n\n[HTML]\n%s", textStr, htmlStr)

	screenshot, err := page.Screenshot(true, &proto.PageCaptureScreenshot{Format: "png"})
	if err != nil {
		return Observation{}, fmt.Errorf("capture screenshot: %w", err)
	}

	return Observation{
		Goal:       goal,
		URL:        url,
		Title:      titleStr,
		DOM:        domStr,
		Clickables: clickablesJSON,
		Screenshot: base64.StdEncoding.EncodeToString(screenshot),
	}, nil
}

func executeDecision(page *rod.Page, decision *aiassist.Decision) error {
	switch decision.Action {
	case "input":
		if decision.Selector == "" {
			return fmt.Errorf("missing selector for input")
		}
		el, err := page.Element(decision.Selector)
		if err != nil {
			return fmt.Errorf("find input element: %w", err)
		}
		_ = el.Focus()
		if err := el.Input(decision.Value); err == nil {
			return nil
		}
		_, err = el.Eval(`(v) => { this.innerText = v; this.dispatchEvent(new Event('input', { bubbles: true })); }`, decision.Value)
		if err != nil {
			return fmt.Errorf("fallback contenteditable input failed: %w", err)
		}
		return nil
	case "click":
		if decision.Selector == "" {
			return fmt.Errorf("missing selector for click")
		}
		el, err := page.Element(decision.Selector)
		if err != nil {
			return fmt.Errorf("find clickable element: %w", err)
		}
		return el.Click(proto.InputMouseButtonLeft, 1)
	case "wait":
		ms := decision.MS
		if ms <= 0 {
			ms = 1000
		}
		time.Sleep(time.Duration(ms) * time.Millisecond)
		return nil
	default:
		return fmt.Errorf("unsupported action: %s", decision.Action)
	}
}

func executeCachedDecision(page *rod.Page, decision CachedDecision) error {
	switch decision.Action {
	case "input":
		if decision.Selector == "" {
			return fmt.Errorf("missing selector for input")
		}
		el, err := page.Element(decision.Selector)
		if err != nil {
			return fmt.Errorf("find input element: %w", err)
		}
		_ = el.Focus()
		if err := el.Input(decision.Value); err == nil {
			return nil
		}
		_, err = el.Eval(`(v) => { this.innerText = v; this.dispatchEvent(new Event('input', { bubbles: true })); }`, decision.Value)
		if err != nil {
			return fmt.Errorf("fallback contenteditable input failed: %w", err)
		}
		return nil
	case "click":
		if decision.Selector == "" {
			return fmt.Errorf("missing selector for click")
		}
		el, err := page.Element(decision.Selector)
		if err != nil {
			return fmt.Errorf("find clickable element: %w", err)
		}
		return el.Click(proto.InputMouseButtonLeft, 1)
	case "wait":
		ms := decision.MS
		if ms <= 0 {
			ms = 1000
		}
		time.Sleep(time.Duration(ms) * time.Millisecond)
		return nil
	default:
		return fmt.Errorf("unsupported action: %s", decision.Action)
	}
}

func (b *BasePublisher) runAIAssist(ctx context.Context, article Article, resume <-chan struct{}, emit EventEmitter, aiConfig AIPublishConfig) error {
	if !aiConfig.Enabled || aiConfig.BaseURL == "" || aiConfig.APIKey == "" {
		return emitNeedsManual(ctx, b.platform, fmt.Sprintf("%s-%d", b.platform, time.Now().UnixNano()), "AI 发布辅助未配置或已关闭，请手动完成发布后点击继续", resume, emit)
	}

	base := b.getBaseProvider()
	browser, cleanup, err := base.LaunchBrowser(false)
	if err != nil {
		return err
	}
	defer cleanup()
	defer base.Close()

	startURL := config.GetHomeURL(b.platform)
	if startURL == "" {
		startURL = base.GetLoginUrl()
	}
	if startURL == "" {
		startURL = "about:blank"
	}

	page := browser.MustPage(startURL)
	defer page.Close()
	page.MustWaitLoad()

	cdpObserver := NewCDPObserver(browser, page)
	if err := cdpObserver.StartObserving(ctx); err != nil {
		b.logger.Warn(fmt.Sprintf("[AIPublish] Failed to start CDP observer: %v", err))
	}
	defer cdpObserver.StopObserving()

	goal := buildGoal(article)

	if cached, ok := getCachedDecisions(b.platform); ok {
		emit("publish:progress", map[string]string{"platform": b.platform, "message": "命中缓存动作序列，自动执行"})
		for _, cd := range cached {
			if cd.Action == "input" && cd.Value != "" {
				if cd.Value == "{{title}}" {
					cd.Value = article.Title
				} else if cd.Value == "{{content}}" {
					cd.Value = article.Content
				}
			}
			if err := executeCachedDecision(page, cd); err != nil {
				clearCachedDecisions(b.platform)
				emit("publish:progress", map[string]string{"platform": b.platform, "message": "缓存动作失效，回退至 AI 决策"})
				return b.runAIAssist(ctx, article, resume, emit, aiConfig)
			}
			waitCtx, waitCancel := context.WithTimeout(ctx, 3*time.Second)
			_ = cdpObserver.WaitForNetworkIdle(waitCtx, 500*time.Millisecond)
			waitCancel()
		}
		return nil
	}

	maxSteps := 12
	decisions := make([]CachedDecision, 0, maxSteps)
	for step := 0; step < maxSteps; step++ {
		enhancedObs, err := cdpObserver.CaptureEnhancedObservation(true)
		if err != nil {
			obs, obsErr := captureObservation(page, goal)
			if obsErr != nil {
				return obsErr
			}
			enhancedObs = &EnhancedObservation{
				URL:         obs.URL,
				Title:       obs.Title,
				VisibleText: obs.DOM,
				Screenshot:  obs.Screenshot,
			}
		}

		clickablesJSON := buildClickablesFromInteractive(enhancedObs.InteractiveElements)
		domStr := buildEnhancedDOM(enhancedObs)

		b.logger.Info(fmt.Sprintf("[AIPublish] observation platform=%s url=%s title=%s forms=%d interactive=%d pending_requests=%d",
			b.platform, enhancedObs.URL, enhancedObs.Title, len(enhancedObs.FormFields), len(enhancedObs.InteractiveElements), len(enhancedObs.PendingRequests)))
		emit("publish:progress", map[string]string{
			"platform": b.platform,
			"message":  fmt.Sprintf("页面观测: forms=%d, buttons=%d, loading=%t", len(enhancedObs.FormFields), len(enhancedObs.InteractiveElements), enhancedObs.IsLoading),
		})

		client, err := aiassist.NewDifyClient(aiConfig.BaseURL, aiConfig.APIKey)
		if err != nil {
			return err
		}
		outputs, err := client.RunWorkflow(ctx, map[string]interface{}{
			"goal":       goal,
			"url":        enhancedObs.URL,
			"title":      enhancedObs.Title,
			"dom":        domStr,
			"clickables": clickablesJSON,
			"screenshot": enhancedObs.Screenshot,
		})
		if err != nil {
			return err
		}
		decision, err := aiassist.ParseDecision(outputs)
		if err != nil {
			return err
		}

		action := strings.ToLower(strings.TrimSpace(decision.Action))
		emit("publish:progress", map[string]string{"platform": b.platform, "message": fmt.Sprintf("AI 指令: %s", action)})

		switch action {
		case "request_manual":
			prompt := decision.Reason
			if prompt == "" {
				prompt = "需要人工操作，请完成后点击继续"
			}
			return emitNeedsManual(ctx, b.platform, fmt.Sprintf("%s-%d", b.platform, time.Now().UnixNano()), prompt, resume, emit)
		case "done":
			if len(decisions) > 0 {
				setCachedDecisions(b.platform, decisions)
			}
			return nil
		case "input", "click", "wait":
			if err := executeDecision(page, decision); err != nil {
				return err
			}
			if action != "wait" {
				waitCtx, waitCancel := context.WithTimeout(ctx, 3*time.Second)
				_ = cdpObserver.WaitForNetworkIdle(waitCtx, 500*time.Millisecond)
				waitCancel()
			}
			val := decision.Value
			if action == "input" {
				if decision.Value == article.Title {
					val = "{{title}}"
				} else if decision.Value == article.Content {
					val = "{{content}}"
				}
			}
			decisions = append(decisions, CachedDecision{
				Action:   action,
				Selector: decision.Selector,
				Value:    val,
				MS:       decision.MS,
			})
		default:
			return fmt.Errorf("unknown action: %s", decision.Action)
		}
	}

	return fmt.Errorf("ai assist exceeded max steps")
}

func buildClickablesFromInteractive(elements []InteractiveElement) string {
	type clickable struct {
		Text     string `json:"text"`
		Selector string `json:"selector"`
		Tag      string `json:"tag"`
	}
	list := make([]clickable, 0, len(elements))
	for _, el := range elements {
		if len(list) >= 60 {
			break
		}
		list = append(list, clickable{
			Text:     el.Text,
			Selector: el.Selector,
			Tag:      el.Tag,
		})
	}
	data, _ := json.Marshal(list)
	return limitString(string(data), 4000)
}

func buildEnhancedDOM(obs *EnhancedObservation) string {
	var sb strings.Builder

	if len(obs.FormFields) > 0 {
		sb.WriteString("[FORM FIELDS]\n")
		for _, f := range obs.FormFields {
			if !f.Visible {
				continue
			}
			sb.WriteString(fmt.Sprintf("- %s (type=%s, selector=%s", f.Label, f.Type, f.Selector))
			if f.Placeholder != "" {
				sb.WriteString(fmt.Sprintf(", placeholder=%s", f.Placeholder))
			}
			if f.Required {
				sb.WriteString(", required")
			}
			sb.WriteString(")\n")
		}
		sb.WriteString("\n")
	}

	sb.WriteString("[TEXT]\n")
	sb.WriteString(limitString(obs.VisibleText, 3000))

	return sb.String()
}

type PopupConfirmPolicy struct {
	ConfirmPattern string
	ExcludePattern string
}

var defaultPopupConfirmPolicy = PopupConfirmPolicy{
	ConfirmPattern: `确定|确认|继续|我知道了|完成|提交`,
	ExcludePattern: `取消|关闭|返回`,
}

var popupConfirmPolicies = map[string]PopupConfirmPolicy{
	"zhihu":     defaultPopupConfirmPolicy,
	"sohu":      defaultPopupConfirmPolicy,
	"csdn":      defaultPopupConfirmPolicy,
	"qie":       defaultPopupConfirmPolicy,
	"baijiahao": defaultPopupConfirmPolicy,
	"toutiao":   defaultPopupConfirmPolicy,
}

func getPopupConfirmPolicy(platform string) PopupConfirmPolicy {
	if policy, ok := popupConfirmPolicies[platform]; ok {
		return policy
	}
	return defaultPopupConfirmPolicy
}

// clickVisibleConfirmButton clicks a visible confirm-style button on popup/dialog.
// It returns whether clicked and the matched button text.
func clickVisibleConfirmButton(page *rod.Page, confirmPattern, excludePattern string, preferDialog bool) (bool, string, error) {
	res, err := page.Eval(`(confirmPattern, excludePattern, preferDialog) => {
		const isVisible = (el) => {
			if (!el) return false;
			const style = window.getComputedStyle(el);
			if (!style || style.visibility === 'hidden' || style.display === 'none') return false;
			const rect = el.getBoundingClientRect();
			return rect.width > 0 && rect.height > 0;
		};
		const norm = (s) => (s || '').replace(/\s+/g, ' ').trim();
		const includeRe = new RegExp(confirmPattern);
		const excludeRe = excludePattern ? new RegExp(excludePattern) : null;

		let roots = [document.body];
		if (preferDialog) {
			const popupRoots = Array.from(document.querySelectorAll('[role="dialog"], .modal, .dialog, .ant-modal, .cheetah-modal-wrap')).filter(isVisible);
			if (popupRoots.length > 0) {
				roots = popupRoots;
			}
		}

		for (const root of roots) {
			const candidates = Array.from(root.querySelectorAll('button, [role="button"], a, span, label')).filter(isVisible);
			for (const el of candidates) {
				const text = norm(el.innerText || el.textContent || el.getAttribute('aria-label') || '');
				if (!text) continue;
				if (excludeRe && excludeRe.test(text)) continue;
				if (!includeRe.test(text)) continue;
				const clickable = el.closest('button, [role="button"], a, label, span') || el;
				clickable.click();
				return { clicked: true, text };
			}
		}

		return { clicked: false, text: '' };
	}`, confirmPattern, excludePattern, preferDialog)
	if err != nil {
		return false, "", err
	}
	return res.Value.Get("clicked").Bool(), res.Value.Get("text").String(), nil
}

func autoConfirmPopupButtons(ctx context.Context, page *rod.Page, confirmPattern, excludePattern string, preferDialog bool, maxRounds int, interval time.Duration) (int, error) {
	count := 0
	for i := 0; i < maxRounds; i++ {
		clicked, _, err := clickVisibleConfirmButton(page, confirmPattern, excludePattern, preferDialog)
		if err != nil {
			return count, err
		}
		if !clicked {
			break
		}
		count++
		select {
		case <-ctx.Done():
			return count, ctx.Err()
		case <-time.After(interval):
		}
	}
	return count, nil
}

func autoConfirmPlatformPopups(ctx context.Context, page *rod.Page, platform string, preferDialog bool, maxRounds int, interval time.Duration) (int, error) {
	policy := getPopupConfirmPolicy(platform)
	return autoConfirmPopupButtons(ctx, page, policy.ConfirmPattern, policy.ExcludePattern, preferDialog, maxRounds, interval)
}

// --- 小红书 ---

type XhsPublisher struct{ *BasePublisher }

func (p *XhsPublisher) Publish(ctx context.Context, article Article, resume <-chan struct{}, emit EventEmitter, aiConfig AIPublishConfig) error {
	log := p.logger

	emit("publish:progress", map[string]string{"platform": "xiaohongshu", "message": "正在打开小红书创作中心..."})
	log.Info("[Xiaohongshu] Starting publish: " + article.Title)

	return p.runAIAssist(ctx, article, resume, emit, aiConfig)
}
