package provider

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/url"
	"reflect"
	"strings"
	"sync"
	"time"

	"geo_client2/backend/config"
	"geo_client2/backend/logger"
	"geo_client2/backend/scrape"

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
	defer page.Close()

	if err := page.WaitLoad(); err != nil {
		d.logger.Warn("[CheckLoginStatus] Yuanbao: WaitLoad failed: " + err.Error())
	}
	rod.Try(func() { _ = page.Timeout(5 * time.Second).WaitStable(1 * time.Second) })
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
	if flow, version, err := scrape.LoadScrapeFlow("yuanbao"); err == nil && flow != nil {
		d.logger.InfoWithContext(ctx, "[YUANBAO-RPA] Loaded scrape flow", map[string]interface{}{"version": version}, nil)
		browser, cleanup, err := d.LaunchBrowser(true)
		if err == nil {
			defer cleanup()
			page := browser.MustPage("")
			defer page.Close()
			runner := scrape.NewRunner(d.logger, "yuanbao")
			vars := map[string]string{"keyword": keyword, "prompt": prompt}
			if runErr := runner.Run(ctx, page, flow, vars); runErr == nil {
				res := runner.Result()
				return &SearchResult{Queries: res.Queries, Citations: convertCitations(res.Citations), FullText: res.FullText}, nil
			}
		}
	}
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

	// Prefer CDP SSE for parsing (avoid double-parse). Fallback to JS hijack if CDP body not available.
	var (
		sawCDPCompleteBody bool
		parseMu            sync.Mutex
	)

	// SSE State Machine - track data labels and event context
	var (
		lastDataLabel string // Track non-JSON data labels like "search_with_text"
		sseStateMu    sync.Mutex
	)

	isLikelyMojibake := func(text string) bool {
		if text == "" {
			return false
		}
		if strings.Contains(text, "Ã") || strings.Contains(text, "æ") || strings.Contains(text, "å") || strings.Contains(text, "ï¼") {
			return true
		}
		return false
	}

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

	// SSE Data Parsing Function with State Machine
	// Yuanbao SSE format:
	//   event: speech_type
	//   data: search_with_text         <- Label indicating next data type
	//   data: {"type":"searchGuid"...} <- Actual citation JSON
	parseSSEData := func(data string) {
		data = strings.TrimSpace(data)

		// Skip event lines and empty data
		if strings.HasPrefix(data, "event:") {
			return
		}
		if strings.HasPrefix(data, "data:") {
			data = strings.TrimSpace(strings.TrimPrefix(data, "data:"))
		}
		if data == "" || data == "[DONE]" || data == "null" {
			return
		}

		sseStateMu.Lock()
		defer sseStateMu.Unlock()

		// Check if this is a JSON object
		isJSON := strings.HasPrefix(data, "{")

		if !isJSON {
			// This is a data label (e.g., "search_with_text", "text", "status")
			lastDataLabel = data
			d.logger.DebugWithContext(ctx, "[YUANBAO-SSE] Data label received", map[string]interface{}{
				"label": data,
			}, nil)
			return
		}

		// This is JSON - parse it
		d.logger.DebugWithContext(ctx, "[YUANBAO-SSE] JSON data received", map[string]interface{}{
			"data_len":      len(data),
			"lastDataLabel": lastDataLabel,
			"data_preview":  data[:min(len(data), 150)],
		}, nil)

		var fullPacket map[string]interface{}
		if err := json.Unmarshal([]byte(data), &fullPacket); err != nil {
			d.logger.DebugWithContext(ctx, "[YUANBAO-SSE] JavaScript fallback JSON unmarshal failed (expected, using CDP Network)", map[string]interface{}{
				"error":        err.Error(),
				"data_len":     len(data),
				"data_preview": data[:min(len(data), 300)],
				"data_suffix":  data[max(0, len(data)-100):],
			}, nil)
			return
		}

		typeField, ok := fullPacket["type"].(string)
		if !ok {
			d.logger.WarnWithContext(ctx, "[YUANBAO-SSE] Missing or invalid 'type' field", map[string]interface{}{
				"keys":         fmt.Sprintf("%v", reflect.ValueOf(fullPacket).MapKeys()),
				"data_preview": data[:min(len(data), 200)],
			}, nil)
			return
		}

		// Process based on type field
		switch typeField {
		case "searchGuid":
			// Citation data: {"type":"searchGuid","docs":[...]}
			var packet struct {
				Type  string `json:"type"`
				Title string `json:"title"`
				Docs  []struct {
					Index       int    `json:"index"`
					DocID       string `json:"docId"`
					Title       string `json:"title"`
					URL         string `json:"url"`
					Quote       string `json:"quote"`
					WebSiteName string `json:"web_site_name"`
					PublishTime string `json:"publish_time"`
				} `json:"docs"`
			}
			if err := json.Unmarshal([]byte(data), &packet); err != nil {
				d.logger.DebugWithContext(ctx, "[YUANBAO-SSE] Citation packet unmarshal failed in JS fallback (will retry via CDP Network)", map[string]interface{}{
					"error":        err.Error(),
					"raw_len":      len(data),
					"data_sample":  data[:min(len(data), 500)],
					"packet_title": fullPacket["title"],
					"docs_count":   len(fullPacket["docs"].([]interface{})),
				}, nil)
				return
			}

			citationMu.Lock()
			beforeCount := len(capturedCitations)
			citationMu.Unlock()

			d.logger.InfoWithContext(ctx, "[YUANBAO-SSE] Processing citations packet", map[string]interface{}{
				"packet_title": packet.Title,
				"docs_count":   len(packet.Docs),
			}, nil)

			for _, item := range packet.Docs {
				if item.URL == "" {
					d.logger.WarnWithContext(ctx, "[YUANBAO-SSE] Skipping doc with empty URL", map[string]interface{}{
						"doc_id":    item.DocID,
						"doc_title": item.Title,
						"index":     item.Index,
					}, nil)
					continue
				}

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

				d.logger.DebugWithContext(ctx, "[YUANBAO-SSE] Added citation", map[string]interface{}{
					"index":  item.Index,
					"url":    item.URL,
					"title":  item.Title[:min(len(item.Title), 50)],
					"domain": cit.Domain,
				}, nil)
			}

			citationMu.Lock()
			afterCount := len(capturedCitations)
			citationMu.Unlock()

			d.logger.InfoWithContext(ctx, "[YUANBAO-SSE] ✅ Citations captured", map[string]interface{}{
				"docs_in_packet":  len(packet.Docs),
				"new_citations":   afterCount - beforeCount,
				"total_citations": afterCount,
				"lastDataLabel":   lastDataLabel,
			}, nil)

			// Reset label after processing
			lastDataLabel = ""

		case "text":
			// Text content: {"type":"text","msg":"..."}
			var packet struct {
				Msg string `json:"msg"`
			}
			if err := json.Unmarshal([]byte(data), &packet); err != nil {
				d.logger.WarnWithContext(ctx, "[YUANBAO-SSE] Text packet unmarshal failed", map[string]interface{}{
					"error": err.Error(),
				}, nil)
				return
			}
			if packet.Msg != "" {
				citationMu.Lock()
				oldLen := len(fullResponseText)
				fullResponseText += packet.Msg
				newLen := len(fullResponseText)
				citationMu.Unlock()

				d.logger.DebugWithContext(ctx, "[YUANBAO-SSE] Text chunk added", map[string]interface{}{
					"chunk_len":   len(packet.Msg),
					"total_len":   newLen,
					"added_bytes": newLen - oldLen,
				}, nil)
			}

			// Reset label after processing
			lastDataLabel = ""

		default:
			// Unknown type - log for debugging
			d.logger.WarnWithContext(ctx, "[YUANBAO-SSE] Unknown JSON type", map[string]interface{}{
				"type":          typeField,
				"lastDataLabel": lastDataLabel,
				"data_preview":  data[:min(len(data), 200)],
				"all_keys":      fmt.Sprintf("%v", reflect.ValueOf(fullPacket).MapKeys()),
			}, nil)
			lastDataLabel = ""
		}
	}

	// 1. Enable Network Domain
	if err := (proto.NetworkEnable{}).Call(page); err != nil {
		d.logger.WarnWithContext(ctx, "[YUANBAO-RPA] Failed to enable network", map[string]interface{}{"error": err.Error()}, nil)
	} else {
		d.logger.InfoWithContext(ctx, "[YUANBAO-RPA] ✅ CDP Network enabled successfully", nil, nil)
	}

	// Track SSE request IDs for fetching complete response bodies
	sseRequestIDs := &sync.Map{} // map[NetworkRequestID]bool

	// Track SSE ResponseReceived events
	go func() {
		page.EachEvent(func(e *proto.NetworkResponseReceived) bool {
			select {
			case <-ctx.Done():
				return true
			default:
			}

			// Detect SSE responses by Content-Type or URL pattern
			if e.Response.MIMEType == "text/event-stream" ||
				strings.Contains(e.Response.URL, "/chat") ||
				strings.Contains(e.Response.URL, "/stream") {
				d.logger.DebugWithContext(ctx, "[YUANBAO-CDP-NETWORK] SSE Response detected", map[string]interface{}{
					"request_id": string(e.RequestID),
					"url":        e.Response.URL,
					"mime_type":  e.Response.MIMEType,
				}, nil)
				sseRequestIDs.Store(string(e.RequestID), true)
			}
			return false
		})()
	}()

	// Track LoadingFinished to fetch complete response body
	go func() {
		page.EachEvent(func(e *proto.NetworkLoadingFinished) bool {
			select {
			case <-ctx.Done():
				return true
			default:
			}

			reqID := string(e.RequestID)
			if _, isSSE := sseRequestIDs.Load(reqID); isSSE {
				d.logger.DebugWithContext(ctx, "[YUANBAO-CDP-NETWORK] SSE LoadingFinished, fetching body", map[string]interface{}{
					"request_id": reqID,
				}, nil)

				// Fetch the complete response body via CDP (with delayed retry if empty)
				body, err := proto.NetworkGetResponseBody{RequestID: e.RequestID}.Call(page)
				if err != nil {
					d.logger.WarnWithContext(ctx, "[YUANBAO-CDP-NETWORK] Failed to get response body", map[string]interface{}{
						"request_id": reqID,
						"error":      err.Error(),
					}, nil)
					return false
				}

				responseBody := body.Body
				if body.Base64Encoded {
					decoded, decodeErr := base64.StdEncoding.DecodeString(body.Body)
					if decodeErr != nil {
						d.logger.WarnWithContext(ctx, "[YUANBAO-CDP-NETWORK] Failed to decode base64 response body", map[string]interface{}{
							"request_id": reqID,
							"error":      decodeErr.Error(),
						}, nil)
					} else {
						responseBody = string(decoded)
					}
				}
				if strings.TrimSpace(responseBody) == "" {
					d.logger.WarnWithContext(ctx, "[YUANBAO-CDP-NETWORK] Empty response body, retrying after delay", map[string]interface{}{
						"request_id": reqID,
					}, nil)
					time.Sleep(400 * time.Millisecond)
					retryBody, retryErr := proto.NetworkGetResponseBody{RequestID: e.RequestID}.Call(page)
					if retryErr != nil {
						d.logger.WarnWithContext(ctx, "[YUANBAO-CDP-NETWORK] Retry failed to get response body", map[string]interface{}{
							"request_id": reqID,
							"error":      retryErr.Error(),
						}, nil)
						return false
					}
					retryBodyText := retryBody.Body
					if retryBody.Base64Encoded {
						decoded, decodeErr := base64.StdEncoding.DecodeString(retryBody.Body)
						if decodeErr != nil {
							d.logger.WarnWithContext(ctx, "[YUANBAO-CDP-NETWORK] Failed to decode retry base64 response body", map[string]interface{}{
								"request_id": reqID,
								"error":      decodeErr.Error(),
							}, nil)
						} else {
							retryBodyText = string(decoded)
						}
					}
					if strings.TrimSpace(retryBodyText) != "" {
						responseBody = retryBodyText
					}
				}
				d.logger.InfoWithContext(ctx, "[YUANBAO-CDP-NETWORK] Response body encoding info", map[string]interface{}{
					"request_id":       reqID,
					"base64_encoded":   body.Base64Encoded,
					"body_len":         len(responseBody),
					"body_prefix_hex":  fmt.Sprintf("%x", []byte(responseBody[:min(len(responseBody), 16)])),
					"body_prefix_text": responseBody[:min(len(responseBody), 120)],
				}, nil)
				d.logger.InfoWithContext(ctx, "[YUANBAO-CDP-NETWORK] ✅ Complete SSE body fetched", map[string]interface{}{
					"request_id": reqID,
					"body_len":   len(responseBody),
					"preview":    responseBody[:min(len(responseBody), 300)],
				}, nil)

				if isLikelyMojibake(responseBody) {
					d.logger.WarnWithContext(ctx, "[YUANBAO-CDP-NETWORK] Skipping CDP body parse due to mojibake", map[string]interface{}{
						"request_id": reqID,
						"preview":    responseBody[:min(len(responseBody), 300)],
					}, nil)
					sseRequestIDs.Delete(reqID)
					return false
				}

				citationMu.Lock()
				capturedCitations = nil
				fullResponseText = ""
				citationMu.Unlock()

				parseMu.Lock()
				sawCDPCompleteBody = true
				parseMu.Unlock()

				// Parse all SSE lines from complete body
				lines := strings.Split(responseBody, "\n")
				for _, line := range lines {
					if strings.TrimSpace(line) == "" {
						continue
					}
					d.logger.DebugWithContext(ctx, "[YUANBAO-CDP-NETWORK] Parsing SSE line", map[string]interface{}{
						"request_id":   reqID,
						"line_len":     len(line),
						"line_preview": line[:min(len(line), 120)],
					}, nil)
					parseSSEData(line)
				}

				// Cleanup
				sseRequestIDs.Delete(reqID)
			}
			return false
		})()
	}()

	// Keep original EventSourceMessageReceived as fallback (for real-time partial data)
	go func() {
		page.EachEvent(func(e *proto.NetworkEventSourceMessageReceived) bool {
			select {
			case <-ctx.Done():
				return true
			default:
			}

			rawData := e.Data
			d.logger.DebugWithContext(ctx, "[YUANBAO-CDP-SSE-FALLBACK] Received SSE chunk", map[string]interface{}{
				"data_len":     len(rawData),
				"data_preview": rawData[:min(len(rawData), 150)],
			}, nil)

			parseSSEData(rawData)
			return false
		})()
	}()

	// 2. Bypass Service Worker
	if err := (proto.NetworkSetBypassServiceWorker{Bypass: true}).Call(page); err != nil {
		d.logger.WarnWithContext(ctx, "[YUANBAO-RPA] Failed to set bypass service worker", map[string]interface{}{"error": err.Error()}, nil)
	} else {
		d.logger.InfoWithContext(ctx, "[YUANBAO-RPA] ✅ Service Worker bypass enabled", nil, nil)
	}

	d.logger.InfoWithContext(ctx, "[YUANBAO-RPA] CDP Network listeners activated (ResponseReceived, LoadingFinished, EventSource)", nil, nil)

	hijackScript := `(() => {
			if (window.__YB_HIJACK_INJECTED__) {
			// console output suppressed
			return;
		}
		window.__YB_HIJACK_INJECTED__ = true;
		console.log('__YB_DEBUG__: Hijack script injected v11');
		window.__YB_SEARCHGUID_BUFFER__ = window.__YB_SEARCHGUID_BUFFER__ || [];
		window.__YB_SEARCHGUID_DOCS__ = window.__YB_SEARCHGUID_DOCS__ || [];
		window.__YB_SEARCHGUID_RAW__ = window.__YB_SEARCHGUID_RAW__ || [];
		window.__YB_DETAIL_ITEMS__ = window.__YB_DETAIL_ITEMS__ || [];
			window.__YB_PUSH_SEARCHGUID__ = (line) => {
				try {
					if (!line) return;
					const jsonStr = String(line).replace(/^data:\s*/, '');
					const obj = JSON.parse(jsonStr);
					if (obj && obj.type === 'searchGuid' && Array.isArray(obj.docs)) {
						for (const doc of obj.docs) {
							if (!doc || !doc.url) continue;
							window.__YB_SEARCHGUID_DOCS__.push({
								index: doc.index,
								docId: doc.docId,
								title: doc.title,
								url: doc.url,
								quote: doc.quote,
								web_site_name: doc.web_site_name,
								publish_time: doc.publish_time
							});
						}
						if (window.__YB_SEARCHGUID_DOCS__.length > 500) {
							window.__YB_SEARCHGUID_DOCS__ = window.__YB_SEARCHGUID_DOCS__.slice(-500);
						}
						return;
					}
				} catch (e) {
					// fallthrough to raw buffer
				}
				try {
					window.__YB_SEARCHGUID_RAW__.push(String(line));
					if (window.__YB_SEARCHGUID_RAW__.length > 3) {
						window.__YB_SEARCHGUID_RAW__ = window.__YB_SEARCHGUID_RAW__.slice(-3);
					}
				} catch (e) {
					// ignore
				}
				try {
					window.__YB_SEARCHGUID_BUFFER__.push(line);
					if (window.__YB_SEARCHGUID_BUFFER__.length > 5) {
						window.__YB_SEARCHGUID_BUFFER__ = window.__YB_SEARCHGUID_BUFFER__.slice(-5);
					}
				} catch (e) {
					// ignore
				}
			};
		
		// 1. Hijack Fetch
		const originalFetch = window.fetch;
		window.fetch = async (...args) => {
			const response = await originalFetch(...args);
			
		try {
			const urlStr = args[0] instanceof Request ? args[0].url : String(args[0]);
			// Match Yuanbao API
			if (urlStr.includes('/api/') || urlStr.includes('chat') || urlStr.includes('stream')) {
				const clone = response.clone();
				if (!clone.body) {
					return response;
				}
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
								const events = buffer.split('\n\n');
								buffer = events.pop();
								for (const evt of events) {
									if (!evt.trim()) continue;
									const lines = evt.split('\n');
									let dataPayload = '';
									for (const line of lines) {
										if (line.startsWith('data:')) {
											dataPayload += line.replace(/^data:\s*/, '');
										}
									}
									if (!dataPayload) continue;
									if (dataPayload.includes('"type":"searchGuid"')) {
										window.__YB_PUSH_SEARCHGUID__('data: ' + dataPayload);
										continue;
									}
									console.log('__YB_LINE__:data: ' + dataPayload);
								}
							}
						} catch (e) {
							console.error('__YB_ERROR__:FETCH:' + e.name + ':' + e.message + ':' + e.stack);
						}
					})();
				}
			} catch (err) {
				console.error('__YB_ERROR__:FETCH_OUTER:' + err.name + ':' + err.message + ':' + (err.stack || 'no-stack'));
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
			if (this._ybUrl && (this._ybUrl.includes('chat') || this._ybUrl.includes('/api/') || this._ybUrl.includes('/conversation/'))) {
				let lastSeenLength = 0;
				this.addEventListener('readystatechange', () => {
					if (this.readyState === 3 || this.readyState === 4) {
						try {
							if (this.readyState === 4 && this._ybUrl && this._ybUrl.includes('/conversation/v1/detail')) {
								console.log('__YB_DEBUG__: /conversation/v1/detail response received, length=' + (this.responseText || '').length);
								const detailText = this.responseText;
								if (detailText && detailText.trim().startsWith('{')) {
									try {
										const detailObj = JSON.parse(detailText);
										const convs = detailObj?.data?.convs || detailObj?.convs || detailObj?.data?.data?.convs || [];
										const first = Array.isArray(convs) ? convs[0] : null;
										const content = first?.speechesV2?.content || first?.speeches_v2?.content || [];
										console.log('__YB_DEBUG__: detail API parsed, content items=' + (content ? content.length : 0));
										if (Array.isArray(content) && content.length) {
											let citationCount = 0;
											for (const item of content) {
												if (!item || !item.type) continue;
												window.__YB_DETAIL_ITEMS__.push({
													type: item.type,
													docs: item.docs,
													msg: item.msg
												});
												if (item.type === 'searchGuid' && item.docs) {
													citationCount += item.docs.length;
												}
											}
											console.log('__YB_DEBUG__: Captured ' + citationCount + ' citations from detail API, total items=' + window.__YB_DETAIL_ITEMS__.length);
											if (window.__YB_DETAIL_ITEMS__.length > 200) {
												window.__YB_DETAIL_ITEMS__ = window.__YB_DETAIL_ITEMS__.slice(-200);
											}
										}
									} catch (e) {
										console.error('__YB_ERROR__: Failed to parse detail API: ' + e.message);
									}
								}
							}
							const text = this.responseText;
							if (text) {
								// Only process new content
								const newContent = text.substring(lastSeenLength);
								lastSeenLength = text.length;
								
								const events = newContent.split('\n\n');
								for (const evt of events) {
									if (!evt.trim()) continue;
									const lines = evt.split('\n');
									let dataPayload = '';
									for (const line of lines) {
										if (line.trim() && line.startsWith('data:')) {
											dataPayload += line.replace(/^data:\s*/, '');
										}
									}
									if (!dataPayload) continue;
								console.log('__YB_LINE__:data: ' + dataPayload);
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
					return es;
				};
			}
	})()`

	// Critical: Inject hijack script on new document (runs before page scripts)
	if _, err := page.EvalOnNewDocument(hijackScript); err != nil {
		d.logger.WarnWithContext(ctx, "[YUANBAO-RPA] Failed to inject fetch hijacker (OnNewDocument)", map[string]interface{}{"error": err.Error()}, nil)
	} else {
		d.logger.InfoWithContext(ctx, "[YUANBAO-RPA] ✅ Hijack script registered via EvalOnNewDocument", nil, nil)
	}

	go func() {
		if err := (proto.RuntimeEnable{}).Call(page); err != nil {
			d.logger.WarnWithContext(ctx, "[YUANBAO-SSE] RuntimeEnable failed", map[string]interface{}{"error": err.Error()}, nil)
		}

		var (
			chunkBuffer   string
			chunkBufferMu sync.Mutex
			isChunking    bool
		)

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

				chunkBufferMu.Lock()

				if strings.HasPrefix(logMsg, "__YB_CHUNK_START__:") {
					chunkBuffer = strings.TrimPrefix(logMsg, "__YB_CHUNK_START__:")
					isChunking = true
					d.logger.DebugWithContext(ctx, "[YUANBAO-CHUNK] Chunk started", map[string]interface{}{
						"buffer_len": len(chunkBuffer),
					}, nil)
					chunkBufferMu.Unlock()
					continue
				} else if strings.HasPrefix(logMsg, "__YB_CHUNK_MID__:") {
					if isChunking {
						chunk := strings.TrimPrefix(logMsg, "__YB_CHUNK_MID__:")
						chunkBuffer += chunk
						d.logger.DebugWithContext(ctx, "[YUANBAO-CHUNK] Chunk mid received", map[string]interface{}{
							"chunk_len":  len(chunk),
							"buffer_len": len(chunkBuffer),
						}, nil)
					}
					chunkBufferMu.Unlock()
					continue
				} else if strings.HasPrefix(logMsg, "__YB_CHUNK_END__:") {
					if isChunking {
						chunk := strings.TrimPrefix(logMsg, "__YB_CHUNK_END__:")
						chunkBuffer += chunk
						completeLine := chunkBuffer
						chunkBuffer = ""
						isChunking = false
						chunkBufferMu.Unlock()

						d.logger.InfoWithContext(ctx, "[YUANBAO-CHUNK] Chunk complete - FULL DATA", map[string]interface{}{
							"total_len":       len(completeLine),
							"line_prefix":     completeLine[:min(len(completeLine), 150)],
							"line_suffix":     completeLine[max(0, len(completeLine)-150):],
							"contains_docs":   strings.Contains(completeLine, `"docs":`),
							"contains_url":    strings.Contains(completeLine, `"url":`),
							"contains_search": strings.Contains(completeLine, "searchGuid"),
						}, nil)

						completeLine = strings.TrimSpace(completeLine)
						if completeLine != "" {
							if strings.HasPrefix(completeLine, "event:") {
								parseSSEData(completeLine)
							} else if strings.HasPrefix(completeLine, "data:") {
								if strings.Contains(completeLine, `"type":"searchGuid"`) {
									parseMu.Lock()
									hasCDPCompleteBody := sawCDPCompleteBody
									parseMu.Unlock()
									if hasCDPCompleteBody {
										d.logger.DebugWithContext(ctx, "[YUANBAO-JS] Skipping searchGuid SSE line (CDP complete body already parsed)", map[string]interface{}{
											"line_len":     len(completeLine),
											"line_preview": completeLine[:min(len(completeLine), 120)],
										}, nil)
										continue
									}
									d.logger.InfoWithContext(ctx, "[YUANBAO-JS] CDP body missing, using JS fallback for searchGuid", map[string]interface{}{
										"line_len":     len(completeLine),
										"line_preview": completeLine[:min(len(completeLine), 120)],
									}, nil)
								}
								parseSSEData(completeLine)
							} else {
								d.logger.WarnWithContext(ctx, "[YUANBAO-CHUNK] Unexpected format (no event/data prefix)", map[string]interface{}{
									"line_start": completeLine[:min(len(completeLine), 50)],
								}, nil)
							}
						}
						continue
					}
					chunkBufferMu.Unlock()
					continue
				} else if strings.HasPrefix(logMsg, "__YB_LINE__:") {
					line := strings.TrimPrefix(logMsg, "__YB_LINE__:")
					chunkBufferMu.Unlock()

					line = strings.TrimSpace(line)
					if line == "" {
						continue
					}

					d.logger.DebugWithContext(ctx, "[YUANBAO-LINE] SSE line", map[string]interface{}{
						"line_len":     len(line),
						"line_preview": line[:min(len(line), 100)],
					}, nil)

					parseMu.Lock()
					if sawCDPCompleteBody {
						parseMu.Unlock()
						d.logger.DebugWithContext(ctx, "[YUANBAO-JS] Skipping SSE line (CDP complete body already parsed)", map[string]interface{}{
							"line_len":     len(line),
							"line_preview": line[:min(len(line), 120)],
						}, nil)
						continue
					}
					parseMu.Unlock()

					if strings.HasPrefix(line, "event:") {
						d.logger.DebugWithContext(ctx, "[YUANBAO-JS] Parsing SSE line", map[string]interface{}{
							"line_len":     len(line),
							"line_preview": line[:min(len(line), 120)],
						}, nil)
						parseSSEData(line)
					} else if strings.HasPrefix(line, "data:") {
						if strings.Contains(line, `"type":"searchGuid"`) {
							parseMu.Lock()
							hasCDPCompleteBody := sawCDPCompleteBody
							parseMu.Unlock()
							if hasCDPCompleteBody {
								d.logger.DebugWithContext(ctx, "[YUANBAO-JS] Skipping searchGuid SSE line (CDP complete body already parsed)", map[string]interface{}{
									"line_len":     len(line),
									"line_preview": line[:min(len(line), 120)],
								}, nil)
								continue
							}
							d.logger.InfoWithContext(ctx, "[YUANBAO-JS] CDP body missing, using JS fallback for searchGuid", map[string]interface{}{
								"line_len":     len(line),
								"line_preview": line[:min(len(line), 120)],
							}, nil)
						}
						d.logger.DebugWithContext(ctx, "[YUANBAO-JS] Parsing SSE line", map[string]interface{}{
							"line_len":     len(line),
							"line_preview": line[:min(len(line), 120)],
						}, nil)
						parseSSEData(line)
					}
					continue
				} else if strings.HasPrefix(logMsg, "__YB_DEBUG__:") {
					chunkBufferMu.Unlock()
					debugMsg := strings.TrimPrefix(logMsg, "__YB_DEBUG__:")
					d.logger.InfoWithContext(ctx, "[YUANBAO-JS-DEBUG] "+debugMsg, nil, nil)
					continue
				} else if strings.HasPrefix(logMsg, "__YB_ERROR__:") {
					chunkBufferMu.Unlock()
					errorMsg := strings.TrimPrefix(logMsg, "__YB_ERROR__:")
					d.logger.WarnWithContext(ctx, "[YUANBAO-JS-ERROR] "+errorMsg, nil, nil)
					continue
				} else if strings.HasPrefix(logMsg, "__YB_") {
					chunkBufferMu.Unlock()
					d.logger.DebugWithContext(ctx, "[YUANBAO-JS] "+logMsg, nil, nil)
					continue
				}

				chunkBufferMu.Unlock()
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

	if _, err := page.Eval(hijackScript); err != nil {
		d.logger.WarnWithContext(ctx, "[YUANBAO-RPA] Failed to re-inject fetch hijacker after navigate", map[string]interface{}{"error": err.Error()}, nil)
	} else {
		d.logger.DebugWithContext(ctx, "[YUANBAO-RPA] Hijack script re-injected after page load", nil, nil)
	}

	finalURL := page.MustInfo().URL
	d.logger.DebugWithContext(ctx, "[YUANBAO-RPA] Current page URL after navigation", map[string]interface{}{"url": finalURL}, nil)

	if strings.Contains(finalURL, "login") || strings.Contains(finalURL, "passport") {
		return nil, fmt.Errorf("redirected to login page, account may need re-authentication: %s", finalURL)
	}

	hasInput, _, _ := page.Has(".ql-editor[contenteditable='true'], textarea, [contenteditable='true']")
	if !hasInput {
		pageHTML, _ := page.HTML()
		d.logger.WarnWithContext(ctx, "[YUANBAO-RPA] No input area found, possible login issue", map[string]interface{}{
			"url":         finalURL,
			"htmlPreview": pageHTML[:min(len(pageHTML), 500)],
		}, nil)
		return nil, fmt.Errorf("no input area found after navigation, account may need login")
	}

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

	// Helper function to extract buffered data from JS
	extractBufferedData := func(logPrefix string) {
		// Pull any buffered searchGuid data from JS (bypass console limits)
		if docsRaw, err := page.Eval(`() => {
			try {
				const docs = window.__YB_SEARCHGUID_DOCS__ || [];
				window.__YB_SEARCHGUID_DOCS__ = [];
				return JSON.stringify(docs);
			} catch (e) {
				return "[]";
			}
		}`); err == nil && docsRaw != nil {
			var docs []struct {
				Index       int    `json:"index"`
				DocID       string `json:"docId"`
				Title       string `json:"title"`
				URL         string `json:"url"`
				Quote       string `json:"quote"`
				WebSiteName string `json:"web_site_name"`
				PublishTime string `json:"publish_time"`
			}
			if err := json.Unmarshal([]byte(docsRaw.Value.String()), &docs); err == nil {
				if len(docs) > 0 {
					d.logger.InfoWithContext(ctx, fmt.Sprintf("[%s] Processing buffered searchGuid docs", logPrefix), map[string]interface{}{
						"docs_count": len(docs),
					}, nil)
					citationMu.Lock()
					beforeCount := len(capturedCitations)
					citationMu.Unlock()
					for _, item := range docs {
						if item.URL == "" {
							continue
						}
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
					citationMu.Lock()
					afterCount := len(capturedCitations)
					citationMu.Unlock()
					d.logger.InfoWithContext(ctx, fmt.Sprintf("[%s] ✅ Citations captured from buffered docs", logPrefix), map[string]interface{}{
						"docs_in_buffer": len(docs),
						"new_citations":  afterCount - beforeCount,
						"total":          afterCount,
					}, nil)
				}
			} else if err != nil {
				d.logger.WarnWithContext(ctx, fmt.Sprintf("[%s] Failed to unmarshal searchGuid docs buffer", logPrefix), map[string]interface{}{"error": err.Error()}, nil)
			}
		} else if err != nil {
			d.logger.WarnWithContext(ctx, fmt.Sprintf("[%s] Failed to read searchGuid docs buffer", logPrefix), map[string]interface{}{"error": err.Error()}, nil)
		}
		if bufferRaw, err := page.Eval(`() => {
			try {
				const buf = window.__YB_SEARCHGUID_BUFFER__ || [];
				window.__YB_SEARCHGUID_BUFFER__ = [];
				return JSON.stringify(buf);
			} catch (e) {
				return "[]";
			}
		}`); err == nil && bufferRaw != nil {
			var lines []string
			if err := json.Unmarshal([]byte(bufferRaw.Value.String()), &lines); err == nil {
				for _, line := range lines {
					line = strings.TrimSpace(line)
					if line == "" {
						continue
					}
					d.logger.DebugWithContext(ctx, fmt.Sprintf("[%s] Processing buffered searchGuid", logPrefix), map[string]interface{}{
						"line_len":     len(line),
						"line_preview": line[:min(len(line), 120)],
					}, nil)
					parseSSEData(line)
				}
			} else if err != nil {
				d.logger.WarnWithContext(ctx, fmt.Sprintf("[%s] Failed to unmarshal searchGuid buffer", logPrefix), map[string]interface{}{"error": err.Error()}, nil)
			}
		} else if err != nil {
			d.logger.WarnWithContext(ctx, fmt.Sprintf("[%s] Failed to read searchGuid buffer", logPrefix), map[string]interface{}{"error": err.Error()}, nil)
		}
		// Fallback v2: read conversation detail items captured from /conversation/v1/detail
		if detailRaw, err := page.Eval(`() => {
			try {
				const items = window.__YB_DETAIL_ITEMS__ || [];
				window.__YB_DETAIL_ITEMS__ = [];
				return JSON.stringify(items);
			} catch (e) {
				return "[]";
			}
		}`); err == nil && detailRaw != nil {
			var items []struct {
				Type string `json:"type"`
				Docs []struct {
					Index       int    `json:"index"`
					DocID       string `json:"docId"`
					Title       string `json:"title"`
					URL         string `json:"url"`
					Quote       string `json:"quote"`
					WebSiteName string `json:"web_site_name"`
					PublishTime string `json:"publish_time"`
				} `json:"docs"`
				Msg string `json:"msg"`
			}
			if err := json.Unmarshal([]byte(detailRaw.Value.String()), &items); err == nil {
				if len(items) > 0 {
					d.logger.InfoWithContext(ctx, fmt.Sprintf("[%s] Processing detail items from /conversation/v1/detail", logPrefix), map[string]interface{}{
						"items_count": len(items),
					}, nil)
					for _, item := range items {
						switch item.Type {
						case "searchGuid":
							citationMu.Lock()
							beforeCount := len(capturedCitations)
							citationMu.Unlock()
							for _, doc := range item.Docs {
								if doc.URL == "" {
									continue
								}
								cit := Citation{
									URL:     doc.URL,
									Title:   doc.Title,
									Snippet: doc.Quote,
									Domain:  doc.WebSiteName,
								}
								if u, err := url.Parse(doc.URL); err == nil && cit.Domain == "" {
									cit.Domain = u.Host
								}
								addCitation(cit)
							}
							citationMu.Lock()
							afterCount := len(capturedCitations)
							citationMu.Unlock()
							d.logger.InfoWithContext(ctx, fmt.Sprintf("[%s] ✅ Citations captured from detail items", logPrefix), map[string]interface{}{
								"docs_in_items": len(item.Docs),
								"new_citations": afterCount - beforeCount,
								"total":         afterCount,
							}, nil)
						case "text":
							if item.Msg != "" {
								citationMu.Lock()
								fullResponseText += item.Msg
								citationMu.Unlock()
							}
						}
					}
				}
			} else if err != nil {
				d.logger.WarnWithContext(ctx, fmt.Sprintf("[%s] Failed to unmarshal detail items", logPrefix), map[string]interface{}{"error": err.Error()}, nil)
			}
		} else if err != nil {
			d.logger.WarnWithContext(ctx, fmt.Sprintf("[%s] Failed to read detail items", logPrefix), map[string]interface{}{"error": err.Error()}, nil)
		}
		if rawRaw, err := page.Eval(`() => {
			try {
				const raw = window.__YB_SEARCHGUID_RAW__ || [];
				window.__YB_SEARCHGUID_RAW__ = [];
				return JSON.stringify(raw);
			} catch (e) {
				return "[]";
			}
		}`); err == nil && rawRaw != nil {
			var raws []string
			if err := json.Unmarshal([]byte(rawRaw.Value.String()), &raws); err == nil {
				for _, raw := range raws {
					raw = strings.TrimSpace(raw)
					if raw == "" {
						continue
					}
					d.logger.WarnWithContext(ctx, fmt.Sprintf("[%s] Raw searchGuid captured", logPrefix), map[string]interface{}{
						"raw_len":     len(raw),
						"raw_prefix":  raw[:min(len(raw), 200)],
						"raw_suffix":  raw[max(0, len(raw)-200):],
						"has_docs":    strings.Contains(raw, `"docs"`),
						"has_end":     strings.HasSuffix(strings.TrimSpace(raw), "}"),
						"has_bracket": strings.Contains(raw, "]"),
					}, nil)
				}
			} else if err != nil {
				d.logger.WarnWithContext(ctx, fmt.Sprintf("[%s] Failed to unmarshal searchGuid raw buffer", logPrefix), map[string]interface{}{"error": err.Error()}, nil)
			}
		} else if err != nil {
			d.logger.WarnWithContext(ctx, fmt.Sprintf("[%s] Failed to read searchGuid raw buffer", logPrefix), map[string]interface{}{"error": err.Error()}, nil)
		}
	}

	// First extraction attempt
	extractBufferedData("YUANBAO-JS-FIRST")

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

	d.logger.InfoWithContext(ctx, "[YUANBAO-RPA] Citation capture summary", map[string]interface{}{
		"cdp_sse_citations":    len(finalCitations),
		"streamed_text_length": len(fullResponseText),
		"dom_text_length":      len(fullText),
	}, nil)

	if len(finalCitations) == 0 {
		d.logger.WarnWithContext(ctx, "[YUANBAO-RPA] No citations from CDP/JS interception, trying DOM fallback...", nil, nil)
		domCitations := d.extractCitationsFromDOM(ctx, page)
		if len(domCitations) > 0 {
			finalCitations = domCitations
			d.logger.InfoWithContext(ctx, "[YUANBAO-RPA] DOM fallback SUCCESS", map[string]interface{}{"count": len(finalCitations)}, nil)
		} else {
			d.logger.WarnWithContext(ctx, "[YUANBAO-RPA] DOM fallback FAILED - no citations found", nil, nil)
			d.logger.InfoWithContext(ctx, "[YUANBAO-RPA] 🔄 Triggering REFRESH FALLBACK (refresh + /conversation/v1/detail)", nil, nil)

			// Wait a bit before refresh to ensure any pending requests complete
			time.Sleep(2 * time.Second)

			if err := page.Reload(); err != nil {
				d.logger.WarnWithContext(ctx, "[YUANBAO-RPA] Refresh failed", map[string]interface{}{"error": err.Error()}, nil)
			} else {
				d.logger.InfoWithContext(ctx, "[YUANBAO-RPA] Page reloaded, waiting for load...", nil, nil)
				_ = page.WaitLoad()

				// Verify hijack script injection
				verifyRes, verifyErr := page.Eval(`() => {
					return {
						injected: !!window.__YB_HIJACK_INJECTED__,
						hasDetailItems: !!window.__YB_DETAIL_ITEMS__,
						detailItemsCount: (window.__YB_DETAIL_ITEMS__ || []).length
					};
				}`)
				if verifyErr == nil && verifyRes != nil {
					d.logger.InfoWithContext(ctx, "[YUANBAO-RPA] Hijack script verification", map[string]interface{}{
						"result": verifyRes.Value.String(),
					}, nil)
				}

				// Re-inject hijack script if needed (belt and suspenders)
				if _, err := page.Eval(hijackScript); err != nil {
					d.logger.WarnWithContext(ctx, "[YUANBAO-RPA] Failed to re-inject hijack after refresh", map[string]interface{}{"error": err.Error()}, nil)
				} else {
					d.logger.InfoWithContext(ctx, "[YUANBAO-RPA] ✅ Hijack script re-injected after refresh", nil, nil)
				}

				// Wait for page to stabilize and API requests to complete
				rod.Try(func() { _ = page.Timeout(8 * time.Second).WaitStable(2 * time.Second) })

				// Check current URL
				currentURL := page.MustInfo().URL
				d.logger.InfoWithContext(ctx, "[YUANBAO-RPA] Current URL after refresh", map[string]interface{}{
					"url": currentURL,
				}, nil)

				d.logger.InfoWithContext(ctx, "[YUANBAO-RPA] Waiting for /conversation/v1/detail API...", nil, nil)
				time.Sleep(6 * time.Second) // Give more time for detail API to complete

				// Check buffer status
				bufferCheckRes, bufferCheckErr := page.Eval(`() => {
					return {
						detailItemsCount: (window.__YB_DETAIL_ITEMS__ || []).length,
						searchGuidDocsCount: (window.__YB_SEARCHGUID_DOCS__ || []).length,
						bufferCount: (window.__YB_SEARCHGUID_BUFFER__ || []).length
					};
				}`)
				if bufferCheckErr == nil && bufferCheckRes != nil {
					d.logger.InfoWithContext(ctx, "[YUANBAO-RPA] Buffer status before extraction", map[string]interface{}{
						"result": bufferCheckRes.Value.String(),
					}, nil)
				}

				// Re-extract buffered data after refresh
				d.logger.InfoWithContext(ctx, "[YUANBAO-RPA] 🔍 Extracting buffered data after refresh...", nil, nil)
				extractBufferedData("YUANBAO-REFRESH-FALLBACK")

				// Check if we got citations after refresh
				citationMu.Lock()
				afterRefreshCount := len(capturedCitations)
				if afterRefreshCount > 0 {
					finalCitations = make([]Citation, len(capturedCitations))
					copy(finalCitations, capturedCitations)
				}
				citationMu.Unlock()

				if afterRefreshCount > 0 {
					d.logger.InfoWithContext(ctx, "[YUANBAO-RPA] ✅ REFRESH FALLBACK SUCCESS", map[string]interface{}{
						"citations_found": afterRefreshCount,
					}, nil)
				} else {
					d.logger.WarnWithContext(ctx, "[YUANBAO-RPA] ❌ REFRESH FALLBACK FAILED - still no citations", nil, nil)

					// Last resort: try DOM extraction one more time after refresh
					domCitationsAfterRefresh := d.extractCitationsFromDOM(ctx, page)
					if len(domCitationsAfterRefresh) > 0 {
						finalCitations = domCitationsAfterRefresh
						d.logger.InfoWithContext(ctx, "[YUANBAO-RPA] ✅ DOM extraction after refresh SUCCESS", map[string]interface{}{
							"count": len(finalCitations),
						}, nil)
					}
				}
			}
		}
	} else {
		d.logger.InfoWithContext(ctx, "[YUANBAO-RPA] Citations captured via CDP/JS", map[string]interface{}{"count": len(finalCitations)}, nil)
	}

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

func (d *YuanbaoProvider) extractCitationsFromDOM(ctx context.Context, page *rod.Page) []Citation {
	citationsJS := `() => {
		const potentialContainers = Array.from(document.querySelectorAll(
			'.markdown-body, div[class*="answer"], div[class*="response"], div[class*="message"], div[class*="content"], article, [class*="ChatMessage"]'
		));
		if (!potentialContainers.length) {
			console.log('__YB_DEBUG__: No response containers found');
			return [];
		}
		
		const lastContainer = potentialContainers[potentialContainers.length - 1];
		console.log('__YB_DEBUG__: Found container with class:', lastContainer.className);
		
		const lists = Array.from(lastContainer.querySelectorAll('ol, ul'));
		const citationLinks = [];
		
		for (const list of lists) {
			// Check if previous sibling is a header "References/Sources"
			// Common in Chinese UIs: "参考资料", "搜索结果", "引用"
			let isCitationSection = false;
			let sibling = list.previousElementSibling;
			let lookbackCount = 0;
			while (sibling && lookbackCount < 3) { // Look back a few elements
				if (['H3', 'H4', 'H5', 'P', 'DIV', 'SPAN'].includes(sibling.tagName)) {
					const text = (sibling.innerText || '').trim();
					if (text.includes('参考') || text.includes('来源') || text.includes('引用') || text.includes('Sources') || text.includes('Search')) {
						isCitationSection = true;
						break;
					}
				}
				sibling = sibling.previousElementSibling;
				if (!sibling || sibling.tagName === 'OL' || sibling.tagName === 'UL') break; // Stop at next list
				lookbackCount++;
			}

			// Gather links if it's a citation section OR if links have numeric pattern "[1]"
			const links = Array.from(list.querySelectorAll('a[href^="http"]'));
			if (links.length > 0) {
				// Heuristic: If list items start with numbers or brackets [1], it's likely citations
				const listText = list.innerText;
				const hasNumbering = /\[\d+\]/.test(listText) || /^\d+\./m.test(listText);
				
				if (isCitationSection || hasNumbering) {
					citationLinks.push(...links);
				}
			}
		}

		// 3. Strategy B: Class Name Scavenge (within container only)
		// If explicit lists failed, look for elements with specific classes
		if (citationLinks.length === 0) {
			const labeledLinks = lastContainer.querySelectorAll('a[class*="ref"], a[class*="source"], a[class*="citation"], div[class*="ref"] a, span[class*="ref"] a');
			citationLinks.push(...labeledLinks);
		}
		
		// 4. Strategy C: Sup/Footnote links
		// Common in some UIs: <sup>[1]</sup> or similar inline citations
		if (citationLinks.length === 0) {
			const supLinks = lastContainer.querySelectorAll('sup a, a[href*="#ref"], a[href*="#cite"]');
			citationLinks.push(...supLinks);
		}

		// Deduplicate and Format
		const seen = new Set();
		return citationLinks.map(a => {
			const href = a.href;
			if (seen.has(href)) return null;
			seen.add(href);
			
			// Clean title: remove leading "[1] " or "1. "
			let title = (a.innerText || a.getAttribute('title') || '').trim();
			title = title.replace(/^\[\d+\]\s*/, '').replace(/^\d+\.\s*/, '');
			
			if (!title || title.length < 2) return null; // Skip empty titles

			return {
				title: title,
				url: href,
				snippet: "", // DOM usually doesn't have snippet
				domain: new URL(href).hostname
			};
		}).filter(Boolean);
	}`

	// Execute the JS
	res, err := page.Eval(citationsJS)
	if err != nil {
		d.logger.WarnWithContext(ctx, "[YUANBAO-RPA] Enhanced DOM extraction failed, falling back to basic scan", map[string]interface{}{"error": err.Error()}, nil)
		return d.extractCitationsBasic(ctx, page) // Fallback to original method
	}

	// Parse JSON results
	var citations []Citation
	if err := json.Unmarshal([]byte(res.Value.String()), &citations); err != nil {
		// If unmarshal fails (or if result is not JSON), fallback
		d.logger.WarnWithContext(ctx, "[YUANBAO-RPA] Failed to unmarshal DOM citations", map[string]interface{}{"error": err.Error()}, nil)
		return d.extractCitationsBasic(ctx, page)
	}

	if len(citations) > 0 {
		return citations
	}

	return d.extractCitationsBasic(ctx, page)
}

func (d *YuanbaoProvider) extractCitationsBasic(ctx context.Context, page *rod.Page) []Citation {
	var citations []Citation

	results, err := page.Eval(`() => {
		const containers = Array.from(document.querySelectorAll('.markdown-body, div[class*="answer"], div[class*="response"], article'));
		if (!containers.length) return [];
		const last = containers[containers.length - 1];

		const keywordMatches = (text) => {
			const t = (text || '').trim();
			if (!t) return false;
			return t.includes('参考') || t.includes('来源') || t.includes('引用') || t.toLowerCase().includes('sources');
		};

		const links = [];
		const lists = Array.from(last.querySelectorAll('ol, ul'));
		for (const list of lists) {
			let isCitationSection = false;
			let sibling = list.previousElementSibling;
			for (let i = 0; i < 3 && sibling; i++) {
				const text = sibling.innerText || '';
				if (keywordMatches(text)) {
					isCitationSection = true;
					break;
				}
				sibling = sibling.previousElementSibling;
			}

			const listLinks = Array.from(list.querySelectorAll('a[href^="http"]'));
			const hasNumbering = listLinks.some(a => {
				const text = a.innerText || '';
				const parentText = a.parentElement ? (a.parentElement.innerText || '') : '';
				return /^\s*\[?\d+\]?/.test(text) || /^\s*\[?\d+\]?/.test(parentText);
			});

			if (listLinks.length > 0 && (isCitationSection || hasNumbering)) {
				links.push(...listLinks);
			}
		}

		if (links.length === 0) {
			const labeled = last.querySelectorAll('a[class*="ref"], a[class*="source"], a[class*="citation"], div[class*="ref"] a');
			links.push(...Array.from(labeled));
		}

		const seen = new Set();
		return links.map(a => {
			const href = a.href;
			if (!href || seen.has(href)) return null;
			seen.add(href);
			const title = (a.innerText || a.getAttribute('title') || '').trim();
			return { url: href, title };
		}).filter(Boolean);
	}`)

	if err == nil && results != nil {
		payload, marshalErr := results.Value.MarshalJSON()
		if marshalErr == nil {
			var items []struct {
				URL   string `json:"url"`
				Title string `json:"title"`
			}
			if jsonErr := json.Unmarshal(payload, &items); jsonErr == nil {
				for _, item := range items {
					if item.URL == "" {
						continue
					}
					cit := Citation{
						URL:          item.URL,
						Title:        strings.TrimSpace(item.Title),
						Snippet:      "",
						QueryIndexes: []int{},
					}
					if u, parseErr := url.Parse(item.URL); parseErr == nil {
						cit.Domain = u.Host
					}
					if cit.Title == "" {
						cit.Title = cit.Domain
					}
					exists := false
					for _, c := range citations {
						if c.URL == cit.URL {
							exists = true
							break
						}
					}
					if !exists {
						citations = append(citations, cit)
					}
				}
			}
		}
	}

	if len(citations) > 0 {
		return citations
	}

	elements, err := page.Elements("a[href^='http']")
	if err != nil {
		return citations
	}

	noiseDomains := []string{
		"yuanbao.tencent.com",
		"tencent.com",
		"qq.com",
		"weixin.qq.com",
		"beian.miit.gov.cn",
		"twitter.com",
		"facebook.com",
		"linkedin.com",
		"instagram.com",
		"discord.com",
		"tiktok.com",
		"weibo.com",
		"gov.cn",
	}

	noiseKeywords := []string{
		"privacy", "terms", "policy", "agreement", "about", "contact",
		"隐私", "条款", "协议", "关于", "联系", "备案",
		"login", "sign up", "登录", "注册",
	}

	for i, el := range elements {
		href, _ := el.Attribute("href")
		if href == nil {
			continue
		}

		hrefStr := strings.ToLower(*href)
		isNoise := false
		for _, nd := range noiseDomains {
			if strings.Contains(hrefStr, nd) {
				isNoise = true
				break
			}
		}
		if isNoise {
			continue
		}

		title, _ := el.Text()
		titleLower := strings.ToLower(title)
		for _, nk := range noiseKeywords {
			if strings.Contains(hrefStr, nk) || strings.Contains(titleLower, nk) {
				isNoise = true
				break
			}
		}
		if isNoise {
			continue
		}

		if len(strings.TrimSpace(title)) < 2 {
			continue
		}

		isSearchResult := false
		class, _ := el.Attribute("class")
		if class != nil && (strings.Contains(*class, "search") || strings.Contains(*class, "result") || strings.Contains(*class, "source") || strings.Contains(*class, "reference") || strings.Contains(*class, "citation")) {
			isSearchResult = true
		}

		if isSearchResult || i < 15 {
			cit := Citation{
				URL:          *href,
				Title:        strings.TrimSpace(title),
				Snippet:      "",
				QueryIndexes: []int{},
			}
			if u, err := url.Parse(*href); err == nil {
				cit.Domain = u.Host
			}

			exists := false
			for _, c := range citations {
				if c.URL == cit.URL {
					exists = true
					break
				}
			}
			if !exists {
				citations = append(citations, cit)
			}
		}

		if len(citations) >= 15 {
			break
		}
	}

	return citations
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
	const maxWaitSeconds = 120
	lastContent := ""
	stableCount := 0
	checkInterval := 1 * time.Second
	startTime := time.Now()

	contentSelectors := []string{
		".markdown-body",
		"div[class*='answer']",
		"div[class*='response']",
		"article",
		"div[class*='message']",
		"div[class*='content']",
		"[class*='ChatMessage']",
	}

	stopButtonSelectors := []string{
		"button[class*='stop']",
		"button[class*='Stop']",
		"button[aria-label*='stop']",
		"button[aria-label*='Stop']",
		"[class*='stop'][class*='button']",
	}

	for {
		select {
		case <-ctx.Done():
			d.logger.InfoWithContext(ctx, "[YUANBAO-RPA] Context cancelled, stopping wait", nil, nil)
			return
		default:
		}

		if time.Since(startTime) > maxWaitSeconds*time.Second {
			d.logger.WarnWithContext(ctx, "[YUANBAO-RPA] waitForResponseComplete reached timeout", map[string]interface{}{
				"maxSeconds":  maxWaitSeconds,
				"finalLength": len(lastContent),
			}, nil)
			return
		}

		time.Sleep(checkInterval)

		currentContent := ""
		for _, sel := range contentSelectors {
			elements, err := page.Elements(sel)
			if err == nil && len(elements) > 0 {
				text, textErr := elements[len(elements)-1].Text()
				if textErr == nil && len(text) > 0 {
					currentContent = text
					break
				}
			}
		}

		if len(currentContent) > 50 {
			if currentContent == lastContent {
				stableCount++
				d.logger.DebugWithContext(ctx, "[YUANBAO-RPA] Content stable", map[string]interface{}{
					"stableCount": stableCount,
					"contentLen":  len(currentContent),
				}, nil)

				if stableCount >= 3 {
					if len(currentContent) < 500 {
						d.logger.WarnWithContext(ctx, "[YUANBAO-RPA] Content too short, likely error page", map[string]interface{}{
							"contentLen":     len(currentContent),
							"contentPreview": currentContent[:min(len(currentContent), 200)],
						}, nil)
						time.Sleep(2 * time.Second)
						stableCount = 0
						continue
					}

					hasStopBtn := false
					for _, sel := range stopButtonSelectors {
						exists, _, _ := page.Has(sel)
						if exists {
							hasStopBtn = true
							break
						}
					}

					if !hasStopBtn {
						hasStopBtnText, _, _ := page.HasR("button, div, span, a", "停止|Stop")
						hasStopBtn = hasStopBtnText
					}

					if !hasStopBtn {
						d.logger.InfoWithContext(ctx, "[YUANBAO-RPA] Response complete (stable + no stop button)", map[string]interface{}{
							"contentLen":     len(currentContent),
							"elapsed":        time.Since(startTime).Seconds(),
							"contentPreview": currentContent[:min(len(currentContent), 150)],
						}, nil)
						return
					}

					d.logger.DebugWithContext(ctx, "[YUANBAO-RPA] Stop button present, response still generating", nil, nil)
					stableCount = 0
					checkInterval = 2 * time.Second
				}
			} else {
				if stableCount > 0 {
					d.logger.DebugWithContext(ctx, "[YUANBAO-RPA] Content changed, resetting stability counter", map[string]interface{}{
						"oldLen": len(lastContent),
						"newLen": len(currentContent),
					}, nil)
				}
				lastContent = currentContent
				stableCount = 0
				checkInterval = 1 * time.Second
			}
		}
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
