package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"sync"
	"time"

	"geo_client2/backend/config"
	"geo_client2/backend/logger"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/input"
	"github.com/go-rod/rod/lib/proto"
)

// YiyanProvider implements Yiyan (Wenxin Yiyan) search with full RPA capabilities.
type YiyanProvider struct {
	*BaseProvider
	logger *logger.Logger
}

// NewYiyanProvider creates a new Yiyan provider.
func NewYiyanProvider(headless bool, timeout int, accountID string) *YiyanProvider {
	return &YiyanProvider{
		BaseProvider: NewBaseProvider("yiyan", headless, timeout, accountID),
		logger:       logger.GetLogger(),
	}
}

// GetLoginUrl returns Yiyan login URL.
func (d *YiyanProvider) GetLoginUrl() string {
	return d.loginURL
}

func (d *YiyanProvider) CheckLoginStatus() (bool, error) {
	browser, cleanup, err := d.LaunchBrowser(true)
	if err != nil {
		return false, err
	}
	defer cleanup()
	defer d.Close()

	page := browser.MustPage(d.GetLoginUrl())
	page.MustWaitLoad()
	time.Sleep(5 * time.Second)

	// Strategy 1: Check Cookies
	cookies, err := page.Cookies([]string{d.GetLoginUrl()})
	if err == nil && len(cookies) > 0 {
		hasAuthCookie := false
		for _, cookie := range cookies {
			cookieName := strings.ToLower(cookie.Name)
			if strings.Contains(cookieName, "session") ||
				strings.Contains(cookieName, "token") ||
				strings.Contains(cookieName, "auth") ||
				strings.Contains(cookieName, "access_token") ||
				strings.Contains(cookieName, "user_id") ||
				cookieName == "jsessionid" ||
				cookieName == "sid" ||
				strings.Contains(cookieName, "bduss") {
				hasAuthCookie = true
				d.logger.Debug(fmt.Sprintf("[CheckLoginStatus] Yiyan: Found auth cookie: %s", cookie.Name))
				break
			}
		}
		if hasAuthCookie {
			d.logger.Debug("[CheckLoginStatus] Yiyan: Valid auth cookie found, likely logged in")
		} else {
			d.logger.Debug("[CheckLoginStatus] Yiyan: No auth cookies found, likely not logged in")
		}
	}

	// Strategy 2: URL Redirect Detection
	finalURL := page.MustInfo().URL

	if strings.Contains(finalURL, "yiyan.baidu.com") && !strings.Contains(finalURL, "login") {
		hasInput, _, _ := page.Has("div[class*='inputArea'], .editorContainer__U2vI65Bv, div[role='textbox'][contenteditable='true']")
		if hasInput {
			d.logger.Debug("[CheckLoginStatus] Yiyan: Found input area, logged in")
			return true, nil
		}
	}

	// Strategy 3: Negative Element Detection (login button)
	hasLoginBtn, _, _ := page.HasR("button, div, span, a", "登录|Login|Sign")
	if hasLoginBtn {
		d.logger.Debug("[CheckLoginStatus] Yiyan: Found login button, not logged in")
		return false, nil
	}

	// Strategy 4: Positive Element Detection (input area)
	hasInput, _, _ := page.Has("div[class*='inputArea'], .editorContainer__U2vI65Bv, div[role='textbox'][contenteditable='true']")
	if hasInput {
		d.logger.Debug("[CheckLoginStatus] Yiyan: Found input area, logged in")
		return true, nil
	}

	// Strategy 5: HTTP Response Check
	bodyText, err := page.Element("body")
	if err == nil {
		text, _ := bodyText.Text()
		textLower := strings.ToLower(text)

		unauthorizedKeywords := []string{
			"unauthorized", "unauthenticated", "login required", "sign in required",
			"access denied", "invalid session", "token expired",
			"未登录", "请登录", "需要登录", "登录已过期", "未授权", "会话已过期", "令牌无效",
		}

		for _, keyword := range unauthorizedKeywords {
			if strings.Contains(textLower, keyword) {
				d.logger.Debug(fmt.Sprintf("[CheckLoginStatus] Yiyan: Found unauthorized keyword '%s', not logged in", keyword))
				return false, nil
			}
		}
	}

	// If we reach here with auth cookies, assume logged in
	if err == nil && len(cookies) > 0 {
		for _, cookie := range cookies {
			cookieName := strings.ToLower(cookie.Name)
			if strings.Contains(cookieName, "session") ||
				strings.Contains(cookieName, "token") ||
				strings.Contains(cookieName, "auth") ||
				strings.Contains(cookieName, "bduss") {
				d.logger.Debug("[CheckLoginStatus] Yiyan: Auth cookie present and no negative indicators found, assuming logged in")
				return true, nil
			}
		}
	}

	d.logger.Debug("[CheckLoginStatus] Yiyan: Unclear state, assuming not logged in")
	return false, nil
}

// Search performs a search with full network interception and citation extraction.
func (d *YiyanProvider) Search(ctx context.Context, keyword, prompt string) (*SearchResult, error) {
	d.logger.InfoWithContext(ctx, "[YIYAN-RPA] ========== SEARCH START (Fetch Injection Mode) ==========", map[string]interface{}{
		"keyword":  keyword,
		"prompt":   prompt,
		"headless": d.headless,
	}, nil)

	browser, cleanup, err := d.LaunchBrowser(d.headless)
	if err != nil {
		d.logger.ErrorWithContext(ctx, "[YIYAN-RPA] Failed to launch browser", map[string]interface{}{"error": err.Error()}, err, nil)
		return nil, err
	}
	defer cleanup()

	d.logger.InfoWithContext(ctx, "[YIYAN-RPA] Browser launched successfully", nil, nil)

	homeURL := config.GetHomeURL("yiyan")
	d.logger.InfoWithContext(ctx, "[YIYAN-RPA] Navigating to home URL", map[string]interface{}{"url": homeURL}, nil)

	page := browser.Context(ctx).MustPage()
	defer page.Close()

	// Define data containers
	var capturedCitations []Citation
	var fullResponseText string
	var citationMu sync.Mutex

	// SSE Data Parsing Function
	parseSSEData := func(data string) {
		// Clean prefix
		jsonStr := data
		if strings.HasPrefix(data, "data:") {
			jsonStr = strings.TrimPrefix(data, "data:")
		}
		jsonStr = strings.TrimSpace(jsonStr)

		if jsonStr == "" || jsonStr == "[DONE]" || jsonStr == "null" {
			return
		}

		var packet struct {
			SearchCitations *struct {
				List []struct {
					Title        string `json:"title"`
					URL          string `json:"url"`
					Site         string `json:"site"`
					WildAbstract string `json:"wild_abstract"`
					Index        string `json:"index"`
				} `json:"list"`
			} `json:"searchCitations"`

			Result string `json:"result"`
			Text   string `json:"text"`

			Data *struct {
				TokensAll       string `json:"tokens_all"`
				Answer          string `json:"answer"`
				Content         string `json:"content"`
				IsEnd           int    `json:"is_end"`
				SearchCitations *struct {
					List []struct {
						Title        string `json:"title"`
						URL          string `json:"url"`
						Site         string `json:"site"`
						WildAbstract string `json:"wild_abstract"`
						Index        string `json:"index"`
					} `json:"list"`
				} `json:"searchCitations"`
			} `json:"data"`
		}

		if err := json.Unmarshal([]byte(jsonStr), &packet); err != nil {
			return
		}

		citationMu.Lock()
		defer citationMu.Unlock()

		addCitations := func(list []struct {
			Title        string `json:"title"`
			URL          string `json:"url"`
			Site         string `json:"site"`
			WildAbstract string `json:"wild_abstract"`
			Index        string `json:"index"`
		}) {
			for _, item := range list {
				if item.URL == "" {
					continue
				}
				exists := false
				for _, e := range capturedCitations {
					if e.URL == item.URL {
						exists = true
						break
					}
				}
				if !exists {
					cit := Citation{
						URL:     item.URL,
						Title:   item.Title,
						Snippet: item.WildAbstract,
						Domain:  item.Site,
					}
					if cit.Domain == "" {
						if u, err := url.Parse(item.URL); err == nil {
							cit.Domain = u.Host
						}
					}
					capturedCitations = append(capturedCitations, cit)
				}
			}
		}

		if packet.SearchCitations != nil && len(packet.SearchCitations.List) > 0 {
			addCitations(packet.SearchCitations.List)
		}
		if packet.Data != nil && packet.Data.SearchCitations != nil && len(packet.Data.SearchCitations.List) > 0 {
			addCitations(packet.Data.SearchCitations.List)
		}

		if packet.Result != "" {
			fullResponseText += packet.Result
		}
		if packet.Text != "" {
			fullResponseText += packet.Text
		}

		if packet.Data != nil {
			if packet.Data.TokensAll != "" {
				if len(packet.Data.TokensAll) > len(fullResponseText) {
					fullResponseText = packet.Data.TokensAll
				}
			}
			if packet.Data.Answer != "" {
				if len(packet.Data.Answer) > len(fullResponseText) {
					fullResponseText = packet.Data.Answer
				}
			}
			if packet.Data.Content != "" {
				if len(packet.Data.Content) > len(fullResponseText) {
					fullResponseText = packet.Data.Content
				}
			}
		}
	}

	// 1. Enable Network Domain
	if err := (proto.NetworkEnable{}).Call(page); err != nil {
		d.logger.WarnWithContext(ctx, "[YIYAN-RPA] Failed to enable network", map[string]interface{}{"error": err.Error()}, nil)
	}

	// Start Network Listener
	go func() {
		page.EachEvent(func(e *proto.NetworkEventSourceMessageReceived) bool {
			select {
			case <-ctx.Done():
				return true
			default:
			}
			// Handle SSE data
			parseSSEData(e.Data)
			return false
		})()
	}()

	// 2. Bypass Service Worker
	if err := (proto.NetworkSetBypassServiceWorker{Bypass: true}).Call(page); err != nil {
		d.logger.WarnWithContext(ctx, "[YIYAN-RPA] Failed to set bypass service worker", map[string]interface{}{"error": err.Error()}, nil)
	}

	// Inject Fetch Hijack Script
	hijackScript := `(() => {
		console.log('__YY_DEBUG__: Hijack script injected');
		
		const originalFetch = window.fetch;
		window.fetch = async (...args) => {
			const response = await originalFetch(...args);
			
			try {
				const urlStr = args[0] instanceof Request ? args[0].url : String(args[0]);
				
				// Match Yiyan API
				if (urlStr.includes('/eb/chat/conversation/v2') || urlStr.includes('conversation/v2')) {
					console.log('__YY_REQ__:FETCH:' + urlStr);
					const clone = response.clone();
					const reader = clone.body.getReader();
					const decoder = new TextDecoder();
					
					(async () => {
						try {
							let buffer = '';
							while (true) {
								const { done, value } = await reader.read();
								if (done) break;
								const chunk = decoder.decode(value, { stream: true });
								buffer += chunk;
								
								const lines = buffer.split('\n');
								buffer = lines.pop(); 
								
								for (const line of lines) {
									if (line.trim() === '') continue;
									console.log('__YY_LINE__:' + line);
								}
							}
						} catch (e) {
							console.error('__YY_ERROR__:FETCH:', e.message);
						}
					})();
				}
			} catch (err) {
				console.error('__YY_ERROR__:FETCH_OUTER:', err.message);
			}
			
			return response;
		};
	})()`

	if _, err := page.EvalOnNewDocument(hijackScript); err != nil {
		d.logger.WarnWithContext(ctx, "[YIYAN-RPA] Failed to inject fetch hijacker", map[string]interface{}{"error": err.Error()}, nil)
	}

	// 3. Listen to Console for Hijacked Data
	go func() {
		if err := (proto.RuntimeEnable{}).Call(page); err != nil {
			d.logger.WarnWithContext(ctx, "[YIYAN-SSE] RuntimeEnable failed", map[string]interface{}{"error": err.Error()}, nil)
		}

		page.EachEvent(func(e *proto.RuntimeConsoleAPICalled) bool {
			select {
			case <-ctx.Done():
				return true
			default:
			}

			if e.Type != proto.RuntimeConsoleAPICalledTypeLog {
				return false
			}

			for _, arg := range e.Args {
				j, err := page.ObjectToJSON(arg)
				if err != nil {
					continue
				}
				var logMsg string
				if v := j.Val(); v != nil {
					if ss, ok := v.(string); ok {
						logMsg = ss
					} else {
						logMsg = fmt.Sprint(v)
					}
				}

				if strings.HasPrefix(logMsg, "__YY_LINE__:") {
					line := strings.TrimPrefix(logMsg, "__YY_LINE__:")
					parseSSEData(line)
				}
			}
			return false
		})()
	}()

	// Navigate Home
	page.MustNavigate(homeURL)
	page.MustWaitLoad()

	d.logger.InfoWithContext(ctx, "[YIYAN-RPA] Waiting for page stability...", nil, nil)
	rod.Try(func() {
		page.Timeout(5 * time.Second).WaitStable(1 * time.Second)
	})

	// STEP 1: Find Textarea
	textarea, err := d.findTextarea(ctx, page)
	if err != nil {
		return nil, err
	}

	// STEP 2: Input Keyword
	err = d.inputKeyword(ctx, page, textarea, keyword)
	if err != nil {
		return nil, err
	}

	// STEP 3: Find Submit Button
	submitBtn, err := d.findSubmitButton(ctx, page)
	if err != nil {
		return nil, err
	}

	// STEP 4: Click Submit
	err = d.clickSubmit(ctx, submitBtn, page)
	if err != nil {
		return nil, err
	}

	// STEP 5: Wait for Response
	d.logger.InfoWithContext(ctx, "[YIYAN-RPA] Waiting for AI response...", nil, nil)
	time.Sleep(5 * time.Second)
	d.waitForResponseComplete(ctx, page)

	// STEP 6: Extract Response
	fullText, err := d.extractResponse(ctx, page)
	if err != nil {
		// Log but continue if extraction fails, maybe we got it via SSE
		d.logger.WarnWithContext(ctx, "[YIYAN-RPA] Failed to extract text from DOM", map[string]interface{}{"error": err.Error()}, nil)
	}

	if fullResponseText != "" && len(fullResponseText) > len(fullText) {
		fullText = fullResponseText
	}

	citationMu.Lock()
	finalCitations := make([]Citation, len(capturedCitations))
	copy(finalCitations, capturedCitations)
	citationMu.Unlock()

	return &SearchResult{
		Queries:   []string{}, // Yiyan doesn't seem to provide query suggestions in the example
		Citations: finalCitations,
		FullText:  fullText,
	}, nil
}

func (d *YiyanProvider) findTextarea(ctx context.Context, page *rod.Page) (*rod.Element, error) {
	// Updated selectors based on user feedback
	selectors := []string{
		// New specific selectors
		"div[role='textbox'][contenteditable='true']",
		".editorContainer__U2vI65Bv [contenteditable='true']",
		".inputArea__O5JmW8WL [contenteditable='true']",
		".editable__QRoAFgYA",

		// Fallbacks
		"textarea",
		".input-box textarea",
		"#dialogue-input",
	}

	for _, sel := range selectors {
		elem, err := page.Timeout(3 * time.Second).Element(sel)
		if err == nil && elem != nil {
			d.logger.InfoWithContext(ctx, "[YIYAN-RPA] Textarea found", map[string]interface{}{"selector": sel}, nil)
			return elem, nil
		}
	}
	return nil, fmt.Errorf("textarea not found")
}

func (d *YiyanProvider) inputKeyword(ctx context.Context, page *rod.Page, textarea *rod.Element, keyword string) error {
	// Handle contenteditable div

	// 1. Click to focus
	textarea.MustClick()

	// 2. Clear content (Select All + Backspace)
	page.KeyActions().Press(input.ControlLeft).Type('a').Release(input.ControlLeft).Press(input.Backspace).MustDo()

	// 3. Input text
	// For contenteditable, sometimes MustInput works, but strict typing is safer
	if err := textarea.Input(keyword); err != nil {
		d.logger.WarnWithContext(ctx, "[YIYAN-RPA] Input failed, trying alternative input method", map[string]interface{}{"error": err.Error()}, nil)
		if _, evalErr := textarea.Eval(`(el, text) => {
			el.innerText = text;
			el.dispatchEvent(new Event('input', { bubbles: true }));
		}`, keyword); evalErr != nil {
			return fmt.Errorf("failed to input keyword (fallback): %w", evalErr)
		}
	}

	time.Sleep(500 * time.Millisecond)
	return nil
}

func (d *YiyanProvider) findSubmitButton(ctx context.Context, page *rod.Page) (*rod.Element, error) {
	// Based on user provided HTML
	// <div class="send__fP6LsYgF"><div class="btnContainer__pRAhgzc5">...</div></div>
	selectors := []string{
		// Exact matches from user HTML
		".send__fP6LsYgF",
		".btnContainer__pRAhgzc5",

		// Generic partial matches for resilience
		"div[class*='btnContainer']",
		"div[class*='send'] div[class*='btnContainer']",

		// Fallback
		"button[class*='send']",
		"[aria-label='发送']",
	}

	for _, sel := range selectors {
		elem, err := page.Timeout(2 * time.Second).Element(sel)
		if err == nil && elem != nil {
			d.logger.InfoWithContext(ctx, "[YIYAN-RPA] Submit button found", map[string]interface{}{"selector": sel}, nil)
			return elem, nil
		}
	}

	return nil, fmt.Errorf("submit button not found")
}

func (d *YiyanProvider) clickSubmit(ctx context.Context, submitBtn *rod.Element, page *rod.Page) error {
	submitBtn.MustClick()
	return nil
}

func (d *YiyanProvider) waitForResponseComplete(ctx context.Context, page *rod.Page) {
	// Wait until response stops changing
	// This is a simplified version; production might need more robust checking (like Doubao)
	// For now, just wait a fixed time or check for stop button disappearance

	// Check if "停止生成" (Stop generating) button appears and then disappears
	for i := 0; i < 60; i++ {
		time.Sleep(1 * time.Second)
		// Check for specific stop signal if known, otherwise wait for text stability
	}
}

func (d *YiyanProvider) extractResponse(ctx context.Context, page *rod.Page) (string, error) {
	selectors := []string{
		".markdown-body",
		".answer-content",
		"div[class*='content_'][class*='markdown']",
		"div[class*='answer'] div[class*='content']",
		".answer",
	}

	for _, sel := range selectors {
		elems, err := page.Elements(sel)
		if err == nil && len(elems) > 0 {
			text := elems[len(elems)-1].MustText()
			text = strings.TrimSpace(text)
			if text == "全选" || text == "复制" || text == "重新生成" {
				continue
			}
			if len(text) > 0 {
				return text, nil
			}
		}
	}

	return "", nil
}
