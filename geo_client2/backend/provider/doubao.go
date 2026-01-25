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

// DoubaoProvider implements Doubao search with full RPA capabilities.
type DoubaoProvider struct {
	*BaseProvider
	logger *logger.Logger
}

// NewDoubaoProvider creates a new Doubao provider.
func NewDoubaoProvider(headless bool, timeout int, accountID string) *DoubaoProvider {
	return &DoubaoProvider{
		BaseProvider: NewBaseProvider("doubao", headless, timeout, accountID),
		logger:       logger.GetLogger(),
	}
}

// GetLoginUrl returns Doubao login URL.
func (d *DoubaoProvider) GetLoginUrl() string {
	return d.loginURL
}

func (d *DoubaoProvider) CheckLoginStatus() (bool, error) {
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
				cookieName == "sid" {
				hasAuthCookie = true
				d.logger.Debug(fmt.Sprintf("[CheckLoginStatus] Doubao: Found auth cookie: %s", cookie.Name))
				break
			}
		}
		if hasAuthCookie {
			d.logger.Debug("[CheckLoginStatus] Doubao: Valid auth cookie found, likely logged in")
		} else {
			d.logger.Debug("[CheckLoginStatus] Doubao: No auth cookies found, likely not logged in")
		}
	}

	// Strategy 2: URL Redirect Detection
	finalURL := page.MustInfo().URL

	if strings.Contains(finalURL, "doubao.com") && !strings.Contains(finalURL, "login") && !strings.Contains(finalURL, "sign") {
		hasInput, _, _ := page.Has("textarea, [contenteditable='true']")
		if hasInput {
			d.logger.Debug("[CheckLoginStatus] Doubao: Found input area, logged in")
			return true, nil
		}
	}

	// Strategy 3: Negative Element Detection (login button)
	hasLoginBtn, _, _ := page.HasR("button, div, a", "登录|注册|Login|Sign")
	if hasLoginBtn {
		d.logger.Debug("[CheckLoginStatus] Doubao: Found login button, not logged in")
		return false, nil
	}

	// Strategy 4: Positive Element Detection (input area)
	hasInput, _, _ := page.Has("textarea, [contenteditable='true']")
	if hasInput {
		d.logger.Debug("[CheckLoginStatus] Doubao: Found input area, logged in")
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
				d.logger.Debug(fmt.Sprintf("[CheckLoginStatus] Doubao: Found unauthorized keyword '%s', not logged in", keyword))
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
				strings.Contains(cookieName, "auth") {
				d.logger.Debug("[CheckLoginStatus] Doubao: Auth cookie present and no negative indicators found, assuming logged in")
				return true, nil
			}
		}
	}

	d.logger.Debug("[CheckLoginStatus] Doubao: Unclear state, assuming not logged in")
	return false, nil
}

// Search performs a search with full network interception and citation extraction.
func (d *DoubaoProvider) Search(ctx context.Context, keyword, prompt string) (*SearchResult, error) {
	d.logger.InfoWithContext(ctx, "[DOUBAO-RPA] ========== SEARCH START (Fetch Injection Mode) ==========", map[string]interface{}{
		"keyword":  keyword,
		"prompt":   prompt,
		"headless": d.headless,
	}, nil)

	browser, cleanup, err := d.LaunchBrowser(d.headless)
	if err != nil {
		d.logger.ErrorWithContext(ctx, "[DOUBAO-RPA] Failed to launch browser", map[string]interface{}{"error": err.Error()}, err, nil)
		return nil, err
	}
	defer cleanup()

	d.logger.InfoWithContext(ctx, "[DOUBAO-RPA] Browser launched successfully", nil, nil)

	homeURL := config.GetHomeURL("doubao")
	d.logger.InfoWithContext(ctx, "[DOUBAO-RPA] Navigating to home URL", map[string]interface{}{"url": homeURL}, nil)

	page := browser.Context(ctx).MustPage()
	defer page.Close()

	// 定义数据容器（提前定义，供 Network 和 Console 监听共用）
	var capturedCitations []Citation
	var capturedQueries []string
	var fullResponseText string
	var citationMu sync.Mutex

	// SSE 数据解析函数（供 Network 和 Console 监听共用）
	// 豆包使用 patch_op 结构，与 DeepSeek 的 p/v 结构不同
	parseSSEData := func(data string) {
		// 去掉 "data: " 前缀
		jsonStr := data
		if strings.HasPrefix(data, "data: ") {
			jsonStr = strings.TrimPrefix(data, "data: ")
		}
		jsonStr = strings.TrimSpace(jsonStr)

		if jsonStr == "" || jsonStr == "[DONE]" || jsonStr == "null" {
			return
		}

		// 豆包的主要数据结构
		var packet struct {
			PatchOp []struct {
				PatchObject int             `json:"patch_object"`
				PatchType   int             `json:"patch_type"`
				PatchValue  json.RawMessage `json:"patch_value"`
			} `json:"patch_op"`
			// 备用字段（向后兼容）
			SearchQueries []interface{}   `json:"search_queries"`
			Queries       []interface{}   `json:"queries"`
			SearchResults []interface{}   `json:"search_results"`
			Results       []interface{}   `json:"results"`
			Citations     []interface{}   `json:"citations"`
			Content       string          `json:"content"`
			Text          string          `json:"text"`
			Delta         json.RawMessage `json:"delta"`
			Message       json.RawMessage `json:"message"`
		}

		if err := json.Unmarshal([]byte(jsonStr), &packet); err != nil {
			// 静默忽略解析失败的数据
			return
		}

		citationMu.Lock()
		defer citationMu.Unlock()

		// 辅助函数：添加查询词（去重）
		addQuery := func(queryText string) bool {
			if queryText == "" {
				return false
			}
			for _, eq := range capturedQueries {
				if eq == queryText {
					return false
				}
			}
			capturedQueries = append(capturedQueries, queryText)
			return true
		}

		// 辅助函数：添加引用（去重）
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

		prevQueryCount := len(capturedQueries)
		prevCitationCount := len(capturedCitations)

		// 1. 处理豆包特有的 patch_op 结构
		if len(packet.PatchOp) > 0 {
			for _, patch := range packet.PatchOp {
				// patch_object: 1 表示消息内容
				if patch.PatchObject == 1 && patch.PatchType == 1 {
					var patchValue struct {
						ContentBlock []struct {
							BlockType int `json:"block_type"`
							Content   struct {
								// block_type=10000: 文本块
								TextBlock struct {
									Text string `json:"text"`
								} `json:"text_block"`
								// block_type=10025: 搜索查询结果块
								SearchQueryResultBlock struct {
									Queries []interface{} `json:"queries"`
									Results []struct {
										Index        int   `json:"index"`
										QueryIndexes []int `json:"query_indexes"`
										TextCard     struct {
											URL          string `json:"url"`
											Title        string `json:"title"`
											Summary      string `json:"summary"`
											Sitename     string `json:"sitename"`
											Index        int    `json:"index"`
											QueryIndexes []int  `json:"query_indexes"`
										} `json:"text_card"`
										VideoCard struct {
											URL         string `json:"url"`
											VideoURL    string `json:"video_url"`
											Title       string `json:"title"`
											Description string `json:"description"`
											Summary     string `json:"summary"`
											Platform    string `json:"platform"`
											Index       int    `json:"index"`
										} `json:"video_card"`
									} `json:"results"`
									Summary string `json:"summary"`
								} `json:"search_query_result_block"`
							} `json:"content"`
						} `json:"content_block"`
					}

					if err := json.Unmarshal(patch.PatchValue, &patchValue); err != nil {
						continue
					}

					for _, block := range patchValue.ContentBlock {
						switch block.BlockType {
						case 10000:
							// 文本块
							if block.Content.TextBlock.Text != "" {
								fullResponseText += block.Content.TextBlock.Text
							}

						case 10025:
							// 搜索查询结果块
							searchBlock := block.Content.SearchQueryResultBlock

							// 提取查询词
							for _, q := range searchBlock.Queries {
								switch v := q.(type) {
								case string:
									addQuery(v)
								case map[string]interface{}:
									if query, ok := v["query"].(string); ok && query != "" {
										addQuery(query)
									} else if text, ok := v["text"].(string); ok && text != "" {
										addQuery(text)
									}
								}
							}

							// 提取搜索结果
							for _, r := range searchBlock.Results {
								// 处理 text_card（网页链接）
								if r.TextCard.URL != "" {
									cit := Citation{
										URL:          r.TextCard.URL,
										Title:        r.TextCard.Title,
										Snippet:      r.TextCard.Summary,
										CiteIndex:    r.TextCard.Index,
										QueryIndexes: r.TextCard.QueryIndexes,
									}
									if len(cit.QueryIndexes) == 0 {
										cit.QueryIndexes = r.QueryIndexes
									}
									if cit.CiteIndex == 0 {
										cit.CiteIndex = r.Index
									}
									if u, err := url.Parse(r.TextCard.URL); err == nil {
										cit.Domain = u.Host
									}
									if r.TextCard.Sitename != "" {
										cit.Domain = r.TextCard.Sitename
									}
									addCitation(cit)
								}

								// 处理 video_card（视频链接，如抖音）
								videoURL := r.VideoCard.URL
								if videoURL == "" {
									videoURL = r.VideoCard.VideoURL
								}
								if videoURL != "" {
									title := r.VideoCard.Title
									if title == "" {
										title = r.VideoCard.Description
									}
									snippet := r.VideoCard.Description
									if snippet == "" {
										snippet = r.VideoCard.Summary
									}
									cit := Citation{
										URL:       videoURL,
										Title:     title,
										Snippet:   snippet,
										CiteIndex: r.VideoCard.Index,
									}
									if r.VideoCard.Index == 0 {
										cit.CiteIndex = r.Index
									}
									if u, err := url.Parse(videoURL); err == nil {
										cit.Domain = u.Host
									}
									if r.VideoCard.Platform != "" {
										cit.Domain = r.VideoCard.Platform
									}
									addCitation(cit)
								}
							}
						}
					}
				}
			}
		}

		// 2. 备用结构处理（向后兼容）

		// 处理查询词字段
		processQueries := func(queries []interface{}) {
			for _, q := range queries {
				switch v := q.(type) {
				case string:
					addQuery(v)
				case map[string]interface{}:
					if query, ok := v["query"].(string); ok && query != "" {
						addQuery(query)
					} else if text, ok := v["text"].(string); ok && text != "" {
						addQuery(text)
					}
				}
			}
		}

		if len(packet.SearchQueries) > 0 {
			processQueries(packet.SearchQueries)
		}
		if len(packet.Queries) > 0 {
			processQueries(packet.Queries)
		}

		// 处理搜索结果字段
		processResults := func(results []interface{}) {
			for _, r := range results {
				if rm, ok := r.(map[string]interface{}); ok {
					urlStr, _ := rm["url"].(string)
					if urlStr == "" {
						continue
					}
					cit := Citation{
						URL:     urlStr,
						Title:   getStringField(rm, "title", "name"),
						Snippet: getStringField(rm, "snippet", "content", "description", "summary"),
					}
					if idx, ok := rm["cite_index"].(float64); ok {
						cit.CiteIndex = int(idx)
					} else if idx, ok := rm["index"].(float64); ok {
						cit.CiteIndex = int(idx)
					}
					if u, err := url.Parse(urlStr); err == nil {
						cit.Domain = u.Host
					}
					if siteName := getStringField(rm, "site_name", "source", "domain", "sitename"); siteName != "" {
						cit.Domain = siteName
					}
					addCitation(cit)
				}
			}
		}

		if len(packet.SearchResults) > 0 {
			processResults(packet.SearchResults)
		}
		if len(packet.Results) > 0 {
			processResults(packet.Results)
		}
		if len(packet.Citations) > 0 {
			processResults(packet.Citations)
		}

		// 处理内容字段
		if packet.Content != "" {
			fullResponseText += packet.Content
		}
		if packet.Text != "" {
			fullResponseText += packet.Text
		}

		// 处理 delta 结构
		if len(packet.Delta) > 0 {
			var delta struct {
				Content string `json:"content"`
				Text    string `json:"text"`
			}
			if err := json.Unmarshal(packet.Delta, &delta); err == nil {
				if delta.Content != "" {
					fullResponseText += delta.Content
				}
				if delta.Text != "" {
					fullResponseText += delta.Text
				}
			}
		}

		// 处理 message 结构
		if len(packet.Message) > 0 {
			var msg struct {
				Content string `json:"content"`
			}
			if err := json.Unmarshal(packet.Message, &msg); err == nil {
				if msg.Content != "" {
					fullResponseText += msg.Content
				}
			}
		}

		// 打印日志
		if len(capturedQueries) > prevQueryCount {
			d.logger.InfoWithContext(ctx, "[DOUBAO-SSE] Captured queries", map[string]interface{}{
				"new":     len(capturedQueries) - prevQueryCount,
				"total":   len(capturedQueries),
				"queries": capturedQueries,
			}, nil)
		}
		if len(capturedCitations) > prevCitationCount {
			d.logger.InfoWithContext(ctx, "[DOUBAO-SSE] Captured citations", map[string]interface{}{
				"new":   len(capturedCitations) - prevCitationCount,
				"total": len(capturedCitations),
			}, nil)
		}
	}

	// 1. 启用 Network 域
	if err := (proto.NetworkEnable{}).Call(page); err != nil {
		d.logger.WarnWithContext(ctx, "[DOUBAO-RPA] Failed to enable network", map[string]interface{}{"error": err.Error()}, nil)
	}

	// 启动 Network 事件监听
	go func() {
		var mu sync.Mutex
		requestURLs := make(map[proto.NetworkRequestID]string)

		d.logger.InfoWithContext(ctx, "[DOUBAO-NET] Network listener started", nil, nil)

		page.EachEvent(func(e *proto.NetworkRequestWillBeSent) bool {
			select {
			case <-ctx.Done():
				return true
			default:
			}
			mu.Lock()
			requestURLs[e.RequestID] = e.Request.URL
			mu.Unlock()
			return false
		}, func(e *proto.NetworkRequestWillBeSentExtraInfo) bool {
			select {
			case <-ctx.Done():
				return true
			default:
			}
			return false
		}, func(e *proto.NetworkResponseReceived) bool {
			select {
			case <-ctx.Done():
				return true
			default:
			}
			mu.Lock()
			if _, ok := requestURLs[e.RequestID]; !ok {
				requestURLs[e.RequestID] = e.Response.URL
			}
			mu.Unlock()
			return false
		}, func(e *proto.NetworkResponseReceivedExtraInfo) bool {
			select {
			case <-ctx.Done():
				return true
			default:
			}
			return false
		}, func(e *proto.NetworkDataReceived) bool {
			select {
			case <-ctx.Done():
				return true
			default:
			}
			return false
		}, func(e *proto.NetworkLoadingFinished) bool {
			select {
			case <-ctx.Done():
				return true
			default:
			}
			mu.Lock()
			delete(requestURLs, e.RequestID)
			mu.Unlock()
			return false
		}, func(e *proto.NetworkLoadingFailed) bool {
			select {
			case <-ctx.Done():
				return true
			default:
			}
			mu.Lock()
			delete(requestURLs, e.RequestID)
			mu.Unlock()
			return false
		}, func(e *proto.NetworkWebSocketCreated) bool {
			select {
			case <-ctx.Done():
				return true
			default:
			}
			return false
		}, func(e *proto.NetworkWebSocketHandshakeResponseReceived) bool {
			select {
			case <-ctx.Done():
				return true
			default:
			}
			return false
		}, func(e *proto.NetworkWebSocketFrameReceived) bool {
			select {
			case <-ctx.Done():
				return true
			default:
			}
			return false
		}, func(e *proto.NetworkWebSocketClosed) bool {
			select {
			case <-ctx.Done():
				return true
			default:
			}
			return false
		}, func(e *proto.NetworkEventSourceMessageReceived) bool {
			select {
			case <-ctx.Done():
				return true
			default:
			}
			// 直接处理 SSE 数据
			parseSSEData(e.Data)
			return false
		})()
	}()

	// 2. 强制绕过 Service Worker
	if err := (proto.NetworkSetBypassServiceWorker{Bypass: true}).Call(page); err != nil {
		d.logger.WarnWithContext(ctx, "[DOUBAO-RPA] Failed to set bypass service worker", map[string]interface{}{"error": err.Error()}, nil)
	}

	// 注入 Fetch 和 XHR 劫持脚本（适配豆包 API 端点）
	hijackScript := `(() => {
		console.log('__DB_DEBUG__: Hijack script injected v2');
		
		// 1. 劫持 Fetch
		const originalFetch = window.fetch;
		window.fetch = async (...args) => {
			const response = await originalFetch(...args);
			
			try {
				const urlStr = args[0] instanceof Request ? args[0].url : String(args[0]);
				console.log('__DB_REQ__:FETCH:' + urlStr);
				
				// 豆包 API 端点匹配
				if (urlStr.includes('chat/completion') || urlStr.includes('/api/chat') || 
					urlStr.includes('/api/bot/chat') || urlStr.includes('/stream') ||
					urlStr.includes('/api/v1/chat')) {
					console.log('__DB_DEBUG__: Fetch matched, starting stream read');
					const clone = response.clone();
					const reader = clone.body.getReader();
					const decoder = new TextDecoder();
					
					(async () => {
						try {
							let buffer = '';
							while (true) {
								const { done, value } = await reader.read();
								if (done) {
									console.log('__DB_DEBUG__: Fetch stream done');
									break;
								}
								const chunk = decoder.decode(value, { stream: true });
								buffer += chunk;
								
								const lines = buffer.split('\n');
								buffer = lines.pop(); 
								
								for (const line of lines) {
									if (line.trim() === '') continue;
									console.log('__DB_LINE__:' + line);
								}
							}
						} catch (e) {
							console.error('__DB_ERROR__:FETCH:', e.message);
						}
					})();
				}
			} catch (err) {
				console.error('__DB_ERROR__:FETCH_OUTER:', err.message);
			}
			
			return response;
		};
		
		// 2. 劫持 XMLHttpRequest (备用)
		const originalXHROpen = XMLHttpRequest.prototype.open;
		const originalXHRSend = XMLHttpRequest.prototype.send;
		
		XMLHttpRequest.prototype.open = function(method, url, ...rest) {
			this._dbUrl = url;
			console.log('__DB_REQ__:XHR:' + url);
			return originalXHROpen.call(this, method, url, ...rest);
		};
		
		XMLHttpRequest.prototype.send = function(...args) {
			if (this._dbUrl && (this._dbUrl.includes('chat') || this._dbUrl.includes('/api/'))) {
				console.log('__DB_DEBUG__: XHR matched, attaching listener');
				this.addEventListener('readystatechange', () => {
					if (this.readyState === 3 || this.readyState === 4) {
						try {
							const text = this.responseText;
							if (text) {
								const lines = text.split('\n');
								for (const line of lines) {
									if (line.trim() && line.startsWith('data:')) {
										console.log('__DB_LINE__:' + line);
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
		
		// 3. 劫持 EventSource (如果使用原生 SSE)
		const originalEventSource = window.EventSource;
		if (originalEventSource) {
			window.EventSource = function(url, config) {
				console.log('__DB_REQ__:SSE:' + url);
				const es = new originalEventSource(url, config);
				es.addEventListener('message', (e) => {
					console.log('__DB_LINE__:data: ' + e.data);
				});
				return es;
			};
		}
		
		console.log('__DB_DEBUG__: All hijacks installed');
	})()`

	if _, err := page.EvalOnNewDocument(hijackScript); err != nil {
		d.logger.WarnWithContext(ctx, "[DOUBAO-RPA] Failed to inject fetch hijacker", map[string]interface{}{"error": err.Error()}, nil)
	}

	// 3. 监听 Console 事件处理数据
	go func() {
		// 启用 Runtime 域以接收 console 事件
		if err := (proto.RuntimeEnable{}).Call(page); err != nil {
			d.logger.WarnWithContext(ctx, "[DOUBAO-SSE] RuntimeEnable failed", map[string]interface{}{"error": err.Error()}, nil)
		}

		page.EachEvent(func(e *proto.RuntimeConsoleAPICalled) bool {
			select {
			case <-ctx.Done():
				return true
			default:
			}

			// 处理 console.log 与 console.error
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

				// 只处理 SSE 数据行，忽略其他调试信息
				if strings.HasPrefix(logMsg, "__DB_LINE__:") {
					line := strings.TrimPrefix(logMsg, "__DB_LINE__:")
					line = strings.TrimSpace(line)
					parseSSEData(line)
				} else if strings.HasPrefix(logMsg, "__DB_ERROR__:") {
					// 只打印错误
					d.logger.WarnWithContext(ctx, "[DOUBAO-JS] ERR: "+strings.TrimPrefix(logMsg, "__DB_ERROR__:"), nil, nil)
				}
			}
			return false
		})()
	}()

	// 导航到豆包首页
	page.MustNavigate(homeURL)
	page.MustWaitLoad()

	d.logger.InfoWithContext(ctx, "[DOUBAO-RPA] Waiting for page stability (3s)...", nil, nil)
	rod.Try(func() {
		page.Timeout(3 * time.Second).WaitStable(1 * time.Second)
	})

	d.logPageSnapshot(ctx, page, "AFTER_LOAD")

	// 步骤 1: 开启联网搜索
	d.logger.InfoWithContext(ctx, "[DOUBAO-RPA] ===== STEP 1: Enable Web Search =====", nil, nil)
	webSearchEnabled := d.tryEnableWebSearch(ctx, page)
	d.logger.InfoWithContext(ctx, "[DOUBAO-RPA] Web search toggle result", map[string]interface{}{"enabled": webSearchEnabled}, nil)

	// 步骤 2: 查找输入框
	d.logger.InfoWithContext(ctx, "[DOUBAO-RPA] ===== STEP 2: Find Input Textarea =====", nil, nil)
	textarea, err := d.findTextarea(ctx, page)
	if err != nil {
		return nil, err
	}

	// 步骤 3: 输入关键词
	d.logger.InfoWithContext(ctx, "[DOUBAO-RPA] ===== STEP 3: Input Keyword =====", nil, nil)
	err = d.inputKeyword(ctx, page, textarea, keyword)
	if err != nil {
		return nil, err
	}

	// 步骤 4: 查找发送按钮
	d.logger.InfoWithContext(ctx, "[DOUBAO-RPA] ===== STEP 4: Find Submit Button =====", nil, nil)
	submitBtn, err := d.findSubmitButton(ctx, page)
	if err != nil {
		return nil, err
	}

	// 步骤 5: 点击发送
	d.logger.InfoWithContext(ctx, "[DOUBAO-RPA] ===== STEP 5: Click Submit =====", nil, nil)
	err = d.clickSubmit(ctx, submitBtn, page)
	if err != nil {
		return nil, err
	}

	// 步骤 6: 等待响应
	d.logger.InfoWithContext(ctx, "[DOUBAO-RPA] ===== STEP 6: Wait for Response (via polling) =====", nil, nil)
	d.logger.InfoWithContext(ctx, "[DOUBAO-RPA] Waiting for AI response...", nil, nil)
	time.Sleep(5 * time.Second)

	d.waitForResponseComplete(ctx, page)

	d.logger.InfoWithContext(ctx, "[DOUBAO-RPA] Proceeding to final extraction", nil, nil)
	d.logPageSnapshot(ctx, page, "AFTER_RESPONSE")

	// 步骤 7: 提取响应
	d.logger.InfoWithContext(ctx, "[DOUBAO-RPA] ===== STEP 7: Extract Response =====", nil, nil)
	fullText, err := d.extractResponse(ctx, page)
	if err != nil {
		return nil, err
	}

	// 如果网络拦截获取到了文本，优先使用
	citationMu.Lock()
	if fullResponseText != "" && len(fullResponseText) > len(fullText) {
		fullText = fullResponseText
	}
	citationMu.Unlock()

	d.logger.InfoWithContext(ctx, "[DOUBAO-RPA] ========== SEARCH COMPLETE ==========", map[string]interface{}{
		"response_length":  len(fullText),
		"response_preview": truncateStringDoubao(fullText, 200),
	}, nil)

	citationMu.Lock()
	finalCitations := make([]Citation, len(capturedCitations))
	copy(finalCitations, capturedCitations)
	finalQueries := make([]string, len(capturedQueries))
	copy(finalQueries, capturedQueries)
	citationMu.Unlock()

	// 根据 QueryIndexes[0] 填充每个 Citation 的 Query 字段
	for i := range finalCitations {
		if len(finalCitations[i].QueryIndexes) > 0 {
			idx := finalCitations[i].QueryIndexes[0]
			if idx >= 0 && idx < len(finalQueries) {
				finalCitations[i].Query = finalQueries[idx]
			}
		}
	}

	// DOM 兜底策略：如果没有通过网络拦截获取到引用
	if len(finalCitations) == 0 {
		d.logger.WarnWithContext(ctx, "[DOUBAO-RPA] JS injection missed citations, trying DOM fallback...", nil, nil)
		domCitations := d.extractCitationsFromDOM(ctx, page)
		if len(domCitations) > 0 {
			finalCitations = domCitations
			d.logger.InfoWithContext(ctx, "[DOUBAO-RPA] DOM fallback success", map[string]interface{}{"count": len(finalCitations)}, nil)
		} else {
			d.logger.WarnWithContext(ctx, "[DOUBAO-RPA] DOM fallback also failed", nil, nil)
		}
	}

	return &SearchResult{
		Queries:   finalQueries,
		Citations: finalCitations,
		FullText:  fullText,
	}, nil
}

// checkToggleActive 检查元素及其上下文的激活状态
func (d *DoubaoProvider) checkToggleActive(ctx context.Context, elem *rod.Element) (bool, error) {
	isActive, err := elem.Eval(`() => {
		const el = this;
		
		// 收集检查目标：自身、祖先（3级）、以及这些祖先的所有子孙
		const targets = new Set();
		let current = el;
		for (let i = 0; i < 3 && current; i++) {
			targets.add(current);
			current.querySelectorAll('*').forEach(t => targets.add(t));
			current = current.parentElement;
		}
		
		const activeKeywords = ['active', 'checked', 'selected', 'enabled', 'on', '--checked', '-active'];
		// 豆包可能使用的激活色
		const activeColors = [
			'rgb(77, 107, 254)', 
			'rgb(36, 127, 255)', 
			'#4d6bfe',
			'rgb(24, 144, 255)',
			'rgb(0, 82, 204)',
			'rgb(22, 119, 255)',  // 字节蓝
			'rgb(0, 150, 255)'
		]; 

		for (const t of targets) {
			// 1. 检查 Class
			if (t.className && typeof t.className === 'string') {
				const cls = t.className.toLowerCase();
				if (activeKeywords.some(k => cls.includes(k))) return true;
			}
			
			// 2. 检查属性
			if (t.getAttribute('aria-checked') === 'true') return true;
			if (t.getAttribute('data-state') === 'checked') return true;
			if (t.tagName === 'INPUT' && t.type === 'checkbox' && t.checked) return true;

			// 3. 检查颜色
			const style = window.getComputedStyle(t);
			const props = ['color', 'backgroundColor', 'fill', 'stroke', 'borderColor'];
			
			for (const p of props) {
				const val = style[p];
				if (val && val !== 'none' && val !== 'transparent' && val !== 'rgba(0, 0, 0, 0)') {
					if (activeColors.some(c => val.includes(c))) return true;
					
					// 泛化检查：蓝色系
					if (val.startsWith('rgb')) {
						const parts = val.match(/\d+/g);
						if (parts && parts.length >= 3) {
							const b = parseInt(parts[2]);
							const r = parseInt(parts[0]);
							if (b > 200 && b > r + 50) return true;
						}
					}
				}
			}
		}
		return false;
	}`)

	if err == nil && isActive.Value.Bool() {
		return true, nil
	}

	return false, nil
}

// clickToggleIfInactive 如果开关未激活则点击
func (d *DoubaoProvider) clickToggleIfInactive(ctx context.Context, elem *rod.Element, selector string) bool {
	isActivated, err := d.checkToggleActive(ctx, elem)
	if err != nil {
		d.logger.WarnWithContext(ctx, "[DOUBAO-RPA] State check error", map[string]interface{}{"error": err.Error()}, nil)
	}

	if isActivated {
		d.logger.InfoWithContext(ctx, "[DOUBAO-RPA] Toggle ALREADY ACTIVE, skipping click", map[string]interface{}{
			"selector": selector,
		}, nil)
		return true
	}

	d.logger.InfoWithContext(ctx, "[DOUBAO-RPA] Toggle inactive (or state undetected), performing click...", map[string]interface{}{
		"selector": selector,
	}, nil)

	clickErr := rod.Try(func() {
		elem.ScrollIntoView()
		elem.MustClick()
	})

	if clickErr == nil {
		time.Sleep(1 * time.Second)

		isActivatedNow, _ := d.checkToggleActive(ctx, elem)
		if isActivatedNow {
			d.logger.InfoWithContext(ctx, "[DOUBAO-RPA] Toggle successfully activated", nil, nil)
			return true
		} else {
			d.logger.WarnWithContext(ctx, "[DOUBAO-RPA] Toggle clicked but state check returned false (might be false negative)", nil, nil)
			return true
		}
	}

	d.logger.WarnWithContext(ctx, "[DOUBAO-RPA] Toggle click failed", map[string]interface{}{
		"selector": selector,
		"error":    clickErr.Error(),
	}, nil)
	return false
}

// tryEnableWebSearch 尝试开启联网搜索
func (d *DoubaoProvider) tryEnableWebSearch(ctx context.Context, page *rod.Page) bool {
	// 1. 文本精准匹配
	textPatterns := []string{"联网搜索", "深度搜索", "搜索", "Search the web", "Web Search"}

	for _, pattern := range textPatterns {
		xpath := fmt.Sprintf("//*[self::div or self::button or self::span or self::label][contains(text(), '%s')]", pattern)
		elems, err := page.ElementsX(xpath)

		if err == nil {
			for _, elem := range elems {
				if visible, _ := elem.Visible(); visible {
					d.logger.InfoWithContext(ctx, "[DOUBAO-RPA] Found toggle by text", map[string]interface{}{
						"pattern": pattern,
					}, nil)
					if d.clickToggleIfInactive(ctx, elem, "text:"+pattern) {
						return true
					}
				}
			}
		}
	}

	// 2. 选择器匹配
	selectors := []string{
		"div[class*='search'] input[type='checkbox']",
		"[aria-label*='联网']",
		"[aria-label*='搜索']",
		"[title*='联网']",
		"[title*='搜索']",
		".toggle-button",
		".switch-button",
	}

	for _, sel := range selectors {
		elems, err := page.Timeout(2 * time.Second).Elements(sel)
		if err == nil {
			for _, elem := range elems {
				if visible, _ := elem.Visible(); visible {
					d.logger.InfoWithContext(ctx, "[DOUBAO-RPA] Found potential toggle by selector", map[string]interface{}{
						"selector": sel,
					}, nil)

					if d.clickToggleIfInactive(ctx, elem, sel) {
						return true
					}
				}
			}
		}
	}

	d.logger.WarnWithContext(ctx, "[DOUBAO-RPA] Could not enable web search toggle (no match found)", nil, nil)
	return false
}

// extractCitationsFromDOM 从 DOM 提取引用（兜底策略）
func (d *DoubaoProvider) extractCitationsFromDOM(ctx context.Context, page *rod.Page) []Citation {
	var citations []Citation

	elements, err := page.Elements("a[href^='http']")
	if err != nil {
		return citations
	}

	// 豆包噪音域名
	noiseDomains := []string{
		"doubao.com",
		"bytecheck.com",
		"volcengine.com",
		"bytedance.com",
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

		// 过滤噪音域名
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

		// 过滤噪音关键词
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

		// 必须要有标题
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

			// 去重
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

func (d *DoubaoProvider) logPageSnapshot(ctx context.Context, page *rod.Page, phase string) {
	d.logger.InfoWithContext(ctx, fmt.Sprintf("[DOUBAO-RPA] [%s] Taking page snapshot...", phase), nil, nil)

	url := page.MustInfo().URL
	title := page.MustInfo().Title
	d.logger.InfoWithContext(ctx, fmt.Sprintf("[DOUBAO-RPA] [%s] Page info", phase), map[string]interface{}{
		"url":   url,
		"title": title,
	}, nil)

	selectors := []string{
		"textarea",
		"[contenteditable='true']",
		"div[class*='message']",
		"div[class*='content']",
		"article",
		"div[class*='search']",
		"div[class*='toggle']",
		"input[type='checkbox']",
	}

	for _, sel := range selectors {
		elements, err := page.Elements(sel)
		count := 0
		if err == nil {
			count = len(elements)
		}
		d.logger.InfoWithContext(ctx, fmt.Sprintf("[DOUBAO-RPA] [%s] Selector check", phase), map[string]interface{}{
			"selector": sel,
			"found":    count,
		}, nil)
	}

	textPatterns := []string{"联网", "搜索", "深度", "发送", "停止"}
	for _, pattern := range textPatterns {
		has, elem, _ := page.HasR("*", pattern)
		info := ""
		if has && elem != nil {
			tag, _ := elem.Attribute("tagName")
			class, _ := elem.Attribute("class")
			if tag != nil {
				info = *tag
			}
			if class != nil {
				info += " class=" + truncateStringDoubao(*class, 50)
			}
		}
		d.logger.InfoWithContext(ctx, fmt.Sprintf("[DOUBAO-RPA] [%s] Text pattern check", phase), map[string]interface{}{
			"pattern": pattern,
			"found":   has,
			"element": info,
		}, nil)
	}
}

func (d *DoubaoProvider) findTextarea(ctx context.Context, page *rod.Page) (*rod.Element, error) {
	selectors := []struct {
		selector string
		desc     string
	}{
		{"textarea", "any textarea"},
		{"textarea[placeholder*='输入']", "placeholder contains 输入"},
		{"textarea[placeholder*='提问']", "placeholder contains 提问"},
		{"textarea[placeholder*='Message']", "placeholder contains Message"},
		{"[contenteditable='true']", "contenteditable div"},
		{".input-area textarea", "input-area textarea"},
		{"textarea[class*='input']", "class contains input"},
	}

	for _, s := range selectors {
		d.logger.InfoWithContext(ctx, "[DOUBAO-RPA] Trying textarea selector", map[string]interface{}{
			"selector": s.selector,
			"desc":     s.desc,
		}, nil)

		elem, err := page.Timeout(3 * time.Second).Element(s.selector)
		if err == nil && elem != nil {
			placeholder, _ := elem.Attribute("placeholder")
			class, _ := elem.Attribute("class")
			id, _ := elem.Attribute("id")

			d.logger.InfoWithContext(ctx, "[DOUBAO-RPA] Textarea found!", map[string]interface{}{
				"selector":    s.selector,
				"placeholder": safeStringDoubao(placeholder),
				"class":       truncateStringDoubao(safeStringDoubao(class), 50),
				"id":          safeStringDoubao(id),
			}, nil)
			return elem, nil
		}
		d.logger.InfoWithContext(ctx, "[DOUBAO-RPA] Textarea selector failed", map[string]interface{}{
			"selector": s.selector,
		}, nil)
	}

	return nil, fmt.Errorf("no textarea found with any selector")
}

func (d *DoubaoProvider) inputKeyword(ctx context.Context, page *rod.Page, textarea *rod.Element, keyword string) error {
	d.logger.InfoWithContext(ctx, "[DOUBAO-RPA] Clicking textarea...", nil, nil)
	clickErr := rod.Try(func() {
		textarea.MustClick()
	})
	if clickErr != nil {
		d.logger.WarnWithContext(ctx, "[DOUBAO-RPA] Click failed", map[string]interface{}{"error": clickErr.Error()}, nil)
	}

	d.logger.InfoWithContext(ctx, "[DOUBAO-RPA] Clearing existing text (Ctrl+A, Backspace)...", nil, nil)
	page.KeyActions().Press(input.ControlLeft).Type('a').Release(input.ControlLeft).Press(input.Backspace).MustDo()

	d.logger.InfoWithContext(ctx, "[DOUBAO-RPA] Inputting keyword...", map[string]interface{}{"keyword": keyword}, nil)

	textarea.MustClick()

	if err := textarea.SelectAllText(); err != nil {
		d.logger.WarnWithContext(ctx, "[DOUBAO-RPA] SelectAllText failed", map[string]interface{}{"error": err.Error()}, nil)
	}

	if err := textarea.Input(keyword); err != nil {
		d.logger.WarnWithContext(ctx, "[DOUBAO-RPA] Input failed, trying MustInput", map[string]interface{}{"error": err.Error()}, nil)
		textarea.MustInput(keyword)
	}

	time.Sleep(500 * time.Millisecond)

	value, _ := textarea.Text()
	d.logger.InfoWithContext(ctx, "[DOUBAO-RPA] Textarea value after input", map[string]interface{}{
		"expected": keyword,
		"actual":   value,
		"match":    value == keyword,
	}, nil)

	return nil
}

func (d *DoubaoProvider) findSubmitButton(ctx context.Context, page *rod.Page) (*rod.Element, error) {
	selectors := []struct {
		selector string
		desc     string
	}{
		// 精确匹配（优先）
		{"button#flow-end-msg-send", "button by ID flow-end-msg-send"},
		{"button[data-testid='chat_input_send_button']", "button by data-testid"},
		{".send-btn-wrapper button", "send-btn-wrapper button"},
		// 现有选择器（兜底）
		{"button[type='submit']", "button type submit"},
		{"button:has(svg)", "button with svg"},
		{".send-button", "class send-button"},
		{"[aria-label*='发送']", "aria-label 发送"},
		{"[title*='发送']", "title 发送"},
		{"div[role='button']:has(svg)", "div role button with svg"},
	}

	for _, s := range selectors {
		d.logger.InfoWithContext(ctx, "[DOUBAO-RPA] Trying submit selector", map[string]interface{}{
			"selector": s.selector,
			"desc":     s.desc,
		}, nil)

		elem, err := page.Timeout(2 * time.Second).Element(s.selector)
		if err == nil && elem != nil {
			disabled, _ := elem.Attribute("disabled")
			class, _ := elem.Attribute("class")

			d.logger.InfoWithContext(ctx, "[DOUBAO-RPA] Submit button found!", map[string]interface{}{
				"selector": s.selector,
				"disabled": disabled != nil,
				"class":    truncateStringDoubao(safeStringDoubao(class), 50),
			}, nil)
			return elem, nil
		}
		d.logger.InfoWithContext(ctx, "[DOUBAO-RPA] Submit selector failed", map[string]interface{}{
			"selector": s.selector,
		}, nil)
	}

	d.logger.InfoWithContext(ctx, "[DOUBAO-RPA] Trying text-based submit search...", nil, nil)
	textPatterns := []string{"发送", "Send"}
	for _, pattern := range textPatterns {
		elem, err := page.Timeout(2*time.Second).ElementR("div, button", pattern)
		if err == nil && elem != nil {
			d.logger.InfoWithContext(ctx, "[DOUBAO-RPA] Found submit by text", map[string]interface{}{
				"pattern": pattern,
			}, nil)
			return elem, nil
		}
	}

	return nil, fmt.Errorf("no submit button found with any selector")
}

func (d *DoubaoProvider) clickSubmit(ctx context.Context, submitBtn *rod.Element, page *rod.Page) error {
	disabled, _ := submitBtn.Attribute("disabled")
	if disabled != nil {
		d.logger.WarnWithContext(ctx, "[DOUBAO-RPA] Submit button is disabled!", nil, nil)
	} else {
		d.logger.InfoWithContext(ctx, "[DOUBAO-RPA] Clicking submit button...", nil, nil)
		err := submitBtn.Click(proto.InputMouseButtonLeft, 1)
		if err != nil {
			d.logger.WarnWithContext(ctx, "[DOUBAO-RPA] Click failed, trying MustClick", map[string]interface{}{"error": err.Error()}, nil)
			submitBtn.MustClick()
		}

		time.Sleep(1 * time.Second)
	}

	d.logger.InfoWithContext(ctx, "[DOUBAO-RPA] Sending ENTER key as backup...", nil, nil)
	page.KeyActions().Press(input.Enter).MustDo()

	d.logger.InfoWithContext(ctx, "[DOUBAO-RPA] Submit action completed", nil, nil)
	return nil
}

func (d *DoubaoProvider) waitForResponseComplete(ctx context.Context, page *rod.Page) {
	const maxRetries = 30
	lastContent := ""
	stableCount := 0

	// 豆包可能使用的回答容器选择器
	contentSelectors := []string{
		"article",
		".message-content",
		"[class*='message']",
		"[class*='content']",
		"[class*='answer']",
		"[class*='response']",
		".chat-message",
	}

	for i := 0; i < maxRetries; i++ {
		time.Sleep(2 * time.Second)

		for _, sel := range contentSelectors {
			elements, err := page.Elements(sel)
			if err == nil && len(elements) > 0 {
				lastElem := elements[len(elements)-1]
				currentContent := lastElem.MustText()

				if len(currentContent) > 100 {
					if currentContent == lastContent {
						stableCount++
						if stableCount >= 2 {
							// 检查是否有"停止生成"按钮
							hasStopBtn, _, _ := page.HasR("*", "停止生成")
							if !hasStopBtn {
								hasStopBtn2, _, _ := page.HasR("*", "停止")
								if !hasStopBtn2 {
									d.logger.InfoWithContext(ctx, "[DOUBAO-RPA] Response generation complete", nil, nil)
									return
								}
							}
						}
					} else {
						stableCount = 0
					}

					lastContent = currentContent
					break
				}
			}
		}
	}
}

func (d *DoubaoProvider) extractResponse(ctx context.Context, page *rod.Page) (string, error) {
	responseSelectors := []struct {
		selector string
		desc     string
	}{
		{"article", "article"},
		{"div[class*='message-content']", "message-content"},
		{"div[class*='content']", "content"},
		{"div[class*='markdown']", "markdown"},
		{"div[class*='response']", "response"},
		{"div[class*='answer']", "answer"},
	}

	for _, s := range responseSelectors {
		d.logger.InfoWithContext(ctx, "[DOUBAO-RPA] Trying response selector", map[string]interface{}{
			"selector": s.selector,
			"desc":     s.desc,
		}, nil)

		elements, err := page.Elements(s.selector)
		if err == nil && len(elements) > 0 {
			lastElem := elements[len(elements)-1]
			text := lastElem.MustText()

			d.logger.InfoWithContext(ctx, "[DOUBAO-RPA] Response extracted!", map[string]interface{}{
				"selector":      s.selector,
				"element_count": len(elements),
				"text_length":   len(text),
				"preview":       truncateStringDoubao(text, 100),
			}, nil)

			if len(strings.TrimSpace(text)) > 0 {
				return text, nil
			}
		}
	}

	d.logger.WarnWithContext(ctx, "[DOUBAO-RPA] Falling back to body text", nil, nil)
	bodyText := page.MustElement("body").MustText()
	d.logger.InfoWithContext(ctx, "[DOUBAO-RPA] Body text extracted", map[string]interface{}{
		"length": len(bodyText),
	}, nil)

	return bodyText, nil
}

// 辅助函数

func truncateStringDoubao(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

func safeStringDoubao(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// getStringField 从 map 中获取字符串字段（尝试多个键名）
func getStringField(m map[string]interface{}, keys ...string) string {
	for _, key := range keys {
		if v, ok := m[key].(string); ok && v != "" {
			return v
		}
	}
	return ""
}
