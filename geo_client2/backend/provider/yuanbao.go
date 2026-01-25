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

// YuanbaoProvider implements Tencent Yuanbao search with full RPA capabilities.
type YuanbaoProvider struct {
	*BaseProvider
	logger *logger.Logger
}

// NewYuanbaoProvider creates a new Yuanbao provider.
func NewYuanbaoProvider(headless bool, timeout int, accountID string) *YuanbaoProvider {
	return &YuanbaoProvider{
		BaseProvider: NewBaseProvider("yuanbao", headless, timeout, accountID),
		logger:       logger.GetLogger(),
	}
}

// GetLoginUrl returns Yuanbao login URL.
func (d *YuanbaoProvider) GetLoginUrl() string {
	return d.loginURL
}

func (d *YuanbaoProvider) CheckLoginStatus() (bool, error) {
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
				strings.Contains(cookieName, "uin") {
				hasAuthCookie = true
				d.logger.Debug(fmt.Sprintf("[CheckLoginStatus] Yuanbao: Found auth cookie: %s", cookie.Name))
				break
			}
		}
		if hasAuthCookie {
			d.logger.Debug("[CheckLoginStatus] Yuanbao: Valid auth cookie found, likely logged in")
		} else {
			d.logger.Debug("[CheckLoginStatus] Yuanbao: No auth cookies found, likely not logged in")
		}
	}

	// Strategy 2: URL Redirect Detection
	finalURL := page.MustInfo().URL

	if strings.Contains(finalURL, "yuanbao.tencent.com") && !strings.Contains(finalURL, "login") {
		hasInput, _, _ := page.Has(".ql-editor[contenteditable='true'], textarea, [contenteditable='true']")
		if hasInput {
			d.logger.Debug("[CheckLoginStatus] Yuanbao: Found input area, logged in")
			return true, nil
		}
	}

	// Strategy 3: Negative Element Detection (login button)
	hasLoginBtn, _, _ := page.HasR("button, div, a", "登录|注册|Login|Sign")
	if hasLoginBtn {
		d.logger.Debug("[CheckLoginStatus] Yuanbao: Found login button, not logged in")
		return false, nil
	}

	// Strategy 4: Positive Element Detection (input area)
	hasInput, _, _ := page.Has(".ql-editor[contenteditable='true'], textarea, [contenteditable='true']")
	if hasInput {
		d.logger.Debug("[CheckLoginStatus] Yuanbao: Found input area, logged in")
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
				d.logger.Debug(fmt.Sprintf("[CheckLoginStatus] Yuanbao: Found unauthorized keyword '%s', not logged in", keyword))
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
				strings.Contains(cookieName, "uin") {
				d.logger.Debug("[CheckLoginStatus] Yuanbao: Auth cookie present and no negative indicators found, assuming logged in")
				return true, nil
			}
		}
	}

	d.logger.Debug("[CheckLoginStatus] Yuanbao: Unclear state, assuming not logged in")
	return false, nil
}

// Search performs a search with full network interception and citation extraction.
func (d *YuanbaoProvider) Search(ctx context.Context, keyword, prompt string) (*SearchResult, error) {
	d.logger.InfoWithContext(ctx, "[YUANBAO-RPA] ========== SEARCH START (Fetch Injection Mode) ==========", map[string]interface{}{
		"keyword":  keyword,
		"prompt":   prompt,
		"headless": d.headless,
	}, nil)

	browser, cleanup, err := d.LaunchBrowser(d.headless)
	if err != nil {
		d.logger.ErrorWithContext(ctx, "[YUANBAO-RPA] Failed to launch browser", map[string]interface{}{"error": err.Error()}, err, nil)
		return nil, err
	}
	defer cleanup()

	page := browser.Context(ctx).MustPage()
	defer page.Close()

	// Define data containers
	var capturedCitations []Citation
	var fullResponseText string
	var citationMu sync.Mutex

	// Helper to add citations without duplication
	addCitation := func(cit Citation) {
		if cit.URL == "" {
			return
		}
		citationMu.Lock()
		defer citationMu.Unlock()
		for _, e := range capturedCitations {
			if e.URL == cit.URL {
				return
			}
		}
		capturedCitations = append(capturedCitations, cit)
	}

	// SSE Data Parsing Function - handles CDP-parsed SSE data directly
	// CDP gives us e.Data without "event:" or "data:" prefixes
	// Yuanbao sends: type indicators ("search_with_text", "text") and JSON objects
	parseSSEData := func(data string) {
		data = strings.TrimSpace(data)
		if data == "" || data == "[DONE]" || data == "null" {
			return
		}

		// Skip non-JSON type indicators (e.g., "search_with_text", "text", "status")
		if !strings.HasPrefix(data, "{") {
			return
		}

		// Try to parse as JSON and dispatch based on "type" field
		var typeCheck struct {
			Type string `json:"type"`
		}
		if err := json.Unmarshal([]byte(data), &typeCheck); err != nil {
			return
		}

		switch typeCheck.Type {
		case "searchGuid":
			// Citation data: {"type":"searchGuid","docs":[...]}
			var packet struct {
				Docs []struct {
					Title       string `json:"title"`
					URL         string `json:"url"`
					Quote       string `json:"quote"`
					WebSiteName string `json:"web_site_name"`
				} `json:"docs"`
			}
			if err := json.Unmarshal([]byte(data), &packet); err != nil {
				d.logger.WarnWithContext(ctx, "[YUANBAO-SSE] Citation unmarshal failed", map[string]interface{}{"error": err.Error(), "raw_len": len(data)}, nil)
				return
			}
			for _, item := range packet.Docs {
				cit := Citation{
					URL:     item.URL,
					Title:   item.Title,
					Snippet: item.Quote,
					Domain:  item.WebSiteName,
				}
				if u, err := url.Parse(item.URL); err == nil && cit.Domain == "" {
					cit.Domain = u.Host
				}
				addCitation(cit)
			}
			d.logger.DebugWithContext(ctx, "[YUANBAO-SSE] Captured citations", map[string]interface{}{"count": len(packet.Docs)}, nil)

		case "text":
			// Text content: {"type":"text","msg":"..."}
			var packet struct {
				Msg string `json:"msg"`
			}
			if err := json.Unmarshal([]byte(data), &packet); err != nil {
				return
			}
			if packet.Msg != "" {
				citationMu.Lock()
				fullResponseText += packet.Msg
				citationMu.Unlock()
			}
		}
	}

	// 1. Enable Network Domain
	if err := (proto.NetworkEnable{}).Call(page); err != nil {
		d.logger.WarnWithContext(ctx, "[YUANBAO-RPA] Failed to enable network", map[string]interface{}{"error": err.Error()}, nil)
	}

	// Start Network Listener
	go func() {
		page.EachEvent(func(e *proto.NetworkEventSourceMessageReceived) bool {
			select {
			case <-ctx.Done():
				return true
			default:
			}
			parseSSEData(e.Data)
			return false
		})()
	}()

	// 2. Bypass Service Worker
	if err := (proto.NetworkSetBypassServiceWorker{Bypass: true}).Call(page); err != nil {
		d.logger.WarnWithContext(ctx, "[YUANBAO-RPA] Failed to set bypass service worker", map[string]interface{}{"error": err.Error()}, nil)
	}

	// Inject Fetch/XHR Hijack Script
	hijackScript := `(() => {
		// console.log('__YB_DEBUG__: Hijack script injected v3');
		
		// 1. Hijack Fetch
		const originalFetch = window.fetch;
		window.fetch = async (...args) => {
			const response = await originalFetch(...args);
			
			try {
				const urlStr = args[0] instanceof Request ? args[0].url : String(args[0]);
				
				// Match Yuanbao API
				if (urlStr.includes('/api/') || urlStr.includes('chat') || urlStr.includes('stream')) {
					const clone = response.clone();
					const reader = clone.body.getReader();
					const decoder = new TextDecoder();
					
					(async () => {
						try {
							let buffer = '';
							while (true) {
								const { done, value } = await reader.read();
								if (done) {
									break;
								}
								const chunk = decoder.decode(value, { stream: true });
								buffer += chunk;
								
								const lines = buffer.split('\n');
								buffer = lines.pop(); 
								
								for (const line of lines) {
									if (line.trim() === '') continue;
									console.log('__YB_LINE__:' + line);
								}
							}
						} catch (e) {
							console.error('__YB_ERROR__:FETCH:', e.message);
						}
					})();
				}
			} catch (err) {
				console.error('__YB_ERROR__:FETCH_OUTER:', err.message);
			}
			
			return response;
		};

		// 2. Hijack XMLHttpRequest
		const originalXHROpen = XMLHttpRequest.prototype.open;
		const originalXHRSend = XMLHttpRequest.prototype.send;
		
		XMLHttpRequest.prototype.open = function(method, url, ...rest) {
			this._ybUrl = url;
			return originalXHROpen.call(this, method, url, ...rest);
		};
		
		XMLHttpRequest.prototype.send = function(...args) {
			if (this._ybUrl && (this._ybUrl.includes('chat') || this._ybUrl.includes('/api/'))) {
				let lastSeenLength = 0;
				this.addEventListener('readystatechange', () => {
					if (this.readyState === 3 || this.readyState === 4) {
						try {
							const text = this.responseText;
							if (text) {
								// Only process new content
								const newContent = text.substring(lastSeenLength);
								lastSeenLength = text.length;
								
								const lines = newContent.split('\n');
								for (const line of lines) {
									if (line.trim() && line.startsWith('data:')) {
										console.log('__YB_LINE__:' + line);
									}
								}
							}
						} catch (e) {
							// ignore
						}
					}
				});
			}
			return originalXHRSend.call(this, ...args);
		};

		// 3. Hijack EventSource
		const originalEventSource = window.EventSource;
		if (originalEventSource) {
			window.EventSource = function(url, config) {
				const es = new originalEventSource(url, config);
				es.addEventListener('message', (e) => {
					console.log('__YB_LINE__:data: ' + e.data);
				});
				// Also listen to specific events if needed
				es.addEventListener('text', (e) => console.log('__YB_LINE__:event: text\n' + '__YB_LINE__:data: ' + e.data));
				es.addEventListener('searchGuid', (e) => console.log('__YB_LINE__:event: searchGuid\n' + '__YB_LINE__:data: ' + e.data));
				return es;
			};
		}
	})()`

	if _, err := page.EvalOnNewDocument(hijackScript); err != nil {
		d.logger.WarnWithContext(ctx, "[YUANBAO-RPA] Failed to inject fetch hijacker", map[string]interface{}{"error": err.Error()}, nil)
	}

	// 3. Listen to Console for Hijacked Data
	go func() {
		if err := (proto.RuntimeEnable{}).Call(page); err != nil {
			d.logger.WarnWithContext(ctx, "[YUANBAO-SSE] RuntimeEnable failed", map[string]interface{}{"error": err.Error()}, nil)
		}

		page.EachEvent(func(e *proto.RuntimeConsoleAPICalled) bool {
			select {
			case <-ctx.Done():
				return true
			default:
			}

			if e.Type != proto.RuntimeConsoleAPICalledTypeLog && e.Type != proto.RuntimeConsoleAPICalledTypeError {
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

				if strings.HasPrefix(logMsg, "__YB_LINE__:") {
					line := strings.TrimPrefix(logMsg, "__YB_LINE__:")
					for _, subLine := range strings.Split(line, "\n") {
						subLine = strings.TrimSpace(subLine)
						if subLine == "" {
							continue
						}
						// JS hijack sends raw SSE lines with "data:" prefix
						if strings.HasPrefix(subLine, "data:") {
							data := strings.TrimSpace(strings.TrimPrefix(subLine, "data:"))
							parseSSEData(data)
						}
					}
				} else if strings.HasPrefix(logMsg, "__YB_") {
					// Log all other YB prefixes to the Go logger for debugging
					d.logger.DebugWithContext(ctx, "[YUANBAO-JS] "+logMsg, nil, nil)
				}
			}
			return false
		})()
	}()

	// Navigate Home
	homeURL := config.GetHomeURL("yuanbao")
	if err := page.Navigate(homeURL); err != nil {
		return nil, fmt.Errorf("navigating to home url: %w", err)
	}
	if err := page.WaitLoad(); err != nil {
		return nil, fmt.Errorf("waiting for page load: %w", err)
	}
	rod.Try(func() { _ = page.Timeout(5 * time.Second).WaitStable(1 * time.Second) })

	// STEP 0: Enable Web Search (Toggle)
	d.logger.InfoWithContext(ctx, "[YUANBAO-RPA] Checking Web Search Toggle...", nil, nil)
	// Selector: .yb-internet-search-btn
	// Active class: index_v2_active__mMizI
	// We want to ensure it is active.
	toggleSelector := ".yb-internet-search-btn"
	toggleBtn, err := page.Element(toggleSelector)
	if err == nil && toggleBtn != nil {
		class, _ := toggleBtn.Attribute("class")
		isActive := false
		if class != nil && strings.Contains(*class, "active") {
			isActive = true
		}

		d.logger.DebugWithContext(ctx, "[YUANBAO-RPA] Toggle found", map[string]interface{}{"class": class, "isActive": isActive}, nil)

		if !isActive {
			d.logger.InfoWithContext(ctx, "[YUANBAO-RPA] Toggle inactive, clicking to enable...", nil, nil)
			if err := toggleBtn.Click(proto.InputMouseButtonLeft, 1); err != nil {
				d.logger.WarnWithContext(ctx, "[YUANBAO-RPA] Failed to click toggle", map[string]interface{}{"error": err.Error()}, nil)
			} else {
				time.Sleep(1 * time.Second)
			}
		} else {
			d.logger.DebugWithContext(ctx, "[YUANBAO-RPA] Toggle already active", nil, nil)
		}
	} else {
		d.logger.WarnWithContext(ctx, "[YUANBAO-RPA] Toggle button not found", nil, nil)
	}

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
	d.logger.InfoWithContext(ctx, "[YUANBAO-RPA] Waiting for AI response...", nil, nil)
	d.waitForResponseComplete(ctx, page)

	// STEP 6: Extract Response
	fullText, err := d.extractResponse(ctx, page)
	if err != nil {
		d.logger.WarnWithContext(ctx, "[YUANBAO-RPA] Failed to extract text from DOM", map[string]interface{}{"error": err.Error()}, nil)
	}

	if fullResponseText != "" && len(fullResponseText) > len(fullText) {
		fullText = fullResponseText
	}

	citationMu.Lock()
	finalCitations := make([]Citation, len(capturedCitations))
	copy(finalCitations, capturedCitations)
	citationMu.Unlock()

	d.logger.InfoWithContext(ctx, "[YUANBAO-RPA] Search complete", map[string]interface{}{
		"query":           keyword,
		"citations_found": len(finalCitations),
		"full_text_len":   len(fullText),
	}, nil)

	return &SearchResult{
		Queries:   []string{keyword},
		Citations: finalCitations,
		FullText:  fullText,
	}, nil
}

func (d *YuanbaoProvider) findTextarea(ctx context.Context, page *rod.Page) (*rod.Element, error) {
	selectors := []string{
		".ql-editor[contenteditable='true']",
		// Fallbacks
		"div[role='textbox'][contenteditable='true']",
		"div[contenteditable='true']",
	}

	for _, sel := range selectors {
		elem, err := page.Timeout(3 * time.Second).Element(sel)
		if err == nil && elem != nil {
			d.logger.InfoWithContext(ctx, "[YUANBAO-RPA] Textarea found", map[string]interface{}{"selector": sel}, nil)
			return elem, nil
		}
	}
	return nil, fmt.Errorf("textarea not found")
}

func (d *YuanbaoProvider) inputKeyword(ctx context.Context, page *rod.Page, textarea *rod.Element, keyword string) error {
	textarea.MustClick()
	// Clear existing
	page.KeyActions().Press(input.ControlLeft).Type('a').Release(input.ControlLeft).Press(input.Backspace).MustDo()

	// Input
	if err := textarea.Input(keyword); err != nil {
		d.logger.WarnWithContext(ctx, "[YUANBAO-RPA] Input failed", map[string]interface{}{"error": err.Error()}, nil)
		return fmt.Errorf("input keyword failed: %w", err)
	}

	time.Sleep(500 * time.Millisecond)
	return nil
}

func (d *YuanbaoProvider) findSubmitButton(ctx context.Context, page *rod.Page) (*rod.Element, error) {
	selectors := []string{
		"#yuanbao-send-btn",
		".style__send-btn___RwTm5", // From provided HTML
		"a[class*='send-btn']",
	}

	for _, sel := range selectors {
		elem, err := page.Timeout(2 * time.Second).Element(sel)
		if err == nil && elem != nil {
			d.logger.InfoWithContext(ctx, "[YUANBAO-RPA] Submit button found", map[string]interface{}{"selector": sel}, nil)
			return elem, nil
		}
	}

	return nil, fmt.Errorf("submit button not found")
}

func (d *YuanbaoProvider) clickSubmit(ctx context.Context, submitBtn *rod.Element, page *rod.Page) error {
	submitBtn.MustClick()
	return nil
}

func (d *YuanbaoProvider) waitForResponseComplete(ctx context.Context, page *rod.Page) {
	const maxRetries = 40
	lastContent := ""
	stableCount := 0

	contentSelectors := []string{
		".markdown-body",
		"div[class*='answer']",
		"div[class*='response']",
		"article",
	}

	for i := 0; i < maxRetries; i++ {
		select {
		case <-ctx.Done():
			return
		default:
		}
		time.Sleep(2 * time.Second)

		currentContent := ""
		for _, sel := range contentSelectors {
			elements, err := page.Elements(sel)
			if err == nil && len(elements) > 0 {
				currentContent = elements[len(elements)-1].MustText()
				break
			}
		}

		if currentContent != "" {
			if currentContent == lastContent {
				stableCount++
				if stableCount >= 3 {
					hasStop, _, _ := page.HasR("button, div, span", "停止")
					if !hasStop {
						d.logger.InfoWithContext(ctx, "[YUANBAO-RPA] Response stable and no stop button found. Assuming complete.", nil, nil)
						return
					}
					stableCount = 0
				}
			} else {
				lastContent = currentContent
				stableCount = 0
			}
		}

		submitBtn, err := d.findSubmitButton(ctx, page)
		if err == nil && submitBtn != nil {
			disabled, _ := submitBtn.Attribute("disabled")
			if disabled == nil {
				if len(currentContent) > 50 {
					d.logger.InfoWithContext(ctx, "[YUANBAO-RPA] Submit button enabled and content found. Assuming complete.", nil, nil)
					return
				}
			}
		}
	}
	d.logger.WarnWithContext(ctx, "[YUANBAO-RPA] waitForResponseComplete reached timeout", nil, nil)
}

func (d *YuanbaoProvider) extractResponse(ctx context.Context, page *rod.Page) (string, error) {
	selectors := []string{
		".markdown-body",
		"div[class*='answer']",
		"div[class*='response']",
	}

	for _, sel := range selectors {
		elems, err := page.Elements(sel)
		if err == nil && len(elems) > 0 {
			return elems[len(elems)-1].MustText(), nil
		}
	}

	return "", nil
}
