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

// CheckLoginStatus checks if logged in.
func (d *YuanbaoProvider) CheckLoginStatus() (bool, error) {
	browser, cleanup, err := d.LaunchBrowser(true) // Headless
	if err != nil {
		return false, err
	}
	defer cleanup()
	defer d.Close()

	page := browser.MustPage(d.GetLoginUrl())
	page.MustWaitLoad()

	// Wait a bit for dynamic content
	time.Sleep(3 * time.Second)

	// Check for "登录" (Login) button.
	// Usually "登录" button is present if not logged in.
	hasLoginBtn, _, _ := page.HasR("button, div, span, a", "登录")
	if hasLoginBtn {
		return false, nil
	}

	// Check for input area or specific elements that only appear when logged in
	// Selector from user: .ql-editor[contenteditable='true']
	hasInput, _, _ := page.Has(".ql-editor[contenteditable='true']")
	if hasInput {
		return true, nil
	}

	// If no login button found and we see typical chat elements, assume logged in
	return true, nil
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

	d.logger.InfoWithContext(ctx, "[YUANBAO-RPA] Browser launched successfully", nil, nil)

	homeURL := config.GetHomeURL("yuanbao")
	d.logger.InfoWithContext(ctx, "[YUANBAO-RPA] Navigating to home URL", map[string]interface{}{"url": homeURL}, nil)

	page := browser.Context(ctx).MustPage()

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

		// Yuanbao Structure
		// data: {"type":"searchGuid", "docs":[{"title":"...", "url":"...", "quote":"...", "web_site_name":"..."}], ...}
		var packet struct {
			Type string `json:"type"`
			Docs []struct {
				Title       string `json:"title"`
				URL         string `json:"url"`
				Quote       string `json:"quote"`
				WebSiteName string `json:"web_site_name"`
				Index       int    `json:"index"`
			} `json:"docs"`
			// Content related fields - Yuanbao format for text stream is likely different or in different packet types
			// Assuming generic content field if available, but primarily focusing on docs as requested
			// If text comes in other packets (e.g. type="message"), we would need example data.
			// For now, relying on DOM extraction for full text if SSE doesn't clearly provide it in this packet.
			Content string `json:"content"`
			Delta   string `json:"delta"`
		}

		if err := json.Unmarshal([]byte(jsonStr), &packet); err != nil {
			return
		}

		citationMu.Lock()
		defer citationMu.Unlock()

		addCitation := func(cit Citation) bool {
			if cit.URL == "" {
				return false
			}
			for _, e := range capturedCitations {
				if e.URL == cit.URL {
					return false
				}
			}
			capturedCitations = append(capturedCitations, cit)
			return true
		}

		// Process Docs (Citations)
		if packet.Type == "searchGuid" && len(packet.Docs) > 0 {
			d.logger.InfoWithContext(ctx, "[YUANBAO-SSE] Found search citations", map[string]interface{}{"count": len(packet.Docs)}, nil)
			for _, item := range packet.Docs {
				cit := Citation{
					URL:     item.URL,
					Title:   item.Title,
					Snippet: item.Quote,
					Domain:  item.WebSiteName,
				}

				if item.URL == "" {
					continue
				}

				if cit.Domain == "" {
					if u, err := url.Parse(item.URL); err == nil {
						cit.Domain = u.Host
					}
				}
				addCitation(cit)
			}
		}

		// Process Content (Best guess based on common patterns if not explicit in example)
		if packet.Content != "" {
			fullResponseText += packet.Content
		}
		if packet.Delta != "" {
			fullResponseText += packet.Delta
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
			// Handle SSE data
			parseSSEData(e.Data)
			return false
		})()
	}()

	// 2. Bypass Service Worker
	if err := (proto.NetworkSetBypassServiceWorker{Bypass: true}).Call(page); err != nil {
		d.logger.WarnWithContext(ctx, "[YUANBAO-RPA] Failed to set bypass service worker", map[string]interface{}{"error": err.Error()}, nil)
	}

	// Inject Fetch Hijack Script
	hijackScript := `(() => {
		console.log('__YB_DEBUG__: Hijack script injected');
		
		const originalFetch = window.fetch;
		window.fetch = async (...args) => {
			const response = await originalFetch(...args);
			
			try {
				const urlStr = args[0] instanceof Request ? args[0].url : String(args[0]);
				
				// Match Yuanbao API
				if (urlStr.includes('/api/chat') || urlStr.includes('yuanbao.tencent.com/api/chat')) {
					console.log('__YB_REQ__:FETCH:' + urlStr);
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

				if strings.HasPrefix(logMsg, "__YB_LINE__:") {
					line := strings.TrimPrefix(logMsg, "__YB_LINE__:")
					parseSSEData(line)
				}
			}
			return false
		})()
	}()

	// Navigate Home
	page.MustNavigate(homeURL)
	page.MustWaitLoad()

	d.logger.InfoWithContext(ctx, "[YUANBAO-RPA] Waiting for page stability...", nil, nil)
	rod.Try(func() {
		page.Timeout(5 * time.Second).WaitStable(1 * time.Second)
	})

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

		d.logger.InfoWithContext(ctx, "[YUANBAO-RPA] Toggle found", map[string]interface{}{"class": class, "isActive": isActive}, nil)

		if !isActive {
			d.logger.InfoWithContext(ctx, "[YUANBAO-RPA] Toggle inactive, clicking to enable...", nil, nil)
			if err := toggleBtn.Click(proto.InputMouseButtonLeft, 1); err != nil {
				d.logger.WarnWithContext(ctx, "[YUANBAO-RPA] Failed to click toggle", map[string]interface{}{"error": err.Error()}, nil)
			} else {
				time.Sleep(1 * time.Second)
			}
		} else {
			d.logger.InfoWithContext(ctx, "[YUANBAO-RPA] Toggle already active", nil, nil)
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
	time.Sleep(5 * time.Second)
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

	return &SearchResult{
		Queries:   []string{}, // Yuanbao SSE example didn't show queries
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
	// Simple wait for now
	for i := 0; i < 60; i++ {
		time.Sleep(1 * time.Second)
		// Detect stop button or content stability
	}
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
