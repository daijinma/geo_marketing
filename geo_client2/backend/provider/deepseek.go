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

type DeepSeekProvider struct {
	*BaseProvider
	logger *logger.Logger
}

func NewDeepSeekProvider(headless bool, timeout int, accountID string) *DeepSeekProvider {
	return &DeepSeekProvider{
		BaseProvider: NewBaseProvider("deepseek", headless, timeout, accountID),
		logger:       logger.GetLogger(),
	}
}

func (d *DeepSeekProvider) GetLoginUrl() string {
	return d.loginURL
}

func (d *DeepSeekProvider) CheckLoginStatus() (bool, error) {
	browser, cleanup, err := d.LaunchBrowser(true)
	if err != nil {
		return false, err
	}
	defer cleanup()
	defer d.Close()

	page := browser.MustPage(d.GetLoginUrl())
	page.MustWaitLoad()
	time.Sleep(2 * time.Second)

	hasLoginText, _, _ := page.HasR("div", "Sign in|登录")
	if hasLoginText {
		return false, nil
	}

	return true, nil
}

func (d *DeepSeekProvider) Search(ctx context.Context, keyword, prompt string) (*SearchResult, error) {
	d.logger.InfoWithContext(ctx, "[DEEPSEEK-RPA] ========== SEARCH START (Fetch Injection Mode) ==========", map[string]interface{}{
		"keyword":  keyword,
		"prompt":   prompt,
		"headless": d.headless,
	}, nil)

	browser, cleanup, err := d.LaunchBrowser(d.headless)
	if err != nil {
		d.logger.ErrorWithContext(ctx, "[DEEPSEEK-RPA] Failed to launch browser", map[string]interface{}{"error": err.Error()}, err, nil)
		return nil, err
	}
	defer cleanup()

	d.logger.InfoWithContext(ctx, "[DEEPSEEK-RPA] Browser launched successfully", nil, nil)

	homeURL := config.GetHomeURL("deepseek")
	d.logger.InfoWithContext(ctx, "[DEEPSEEK-RPA] Navigating to home URL", map[string]interface{}{"url": homeURL}, nil)

	page := browser.Context(ctx).MustPage()
	defer page.Close()

	// 定义数据容器（提前定义，供 Network 和 Console 监听共用）
	var capturedCitations []Citation
	var capturedQueries []string
	var citationMu sync.Mutex

	// SSE 数据解析函数（供 Network 和 Console 监听共用）
	parseSSEData := func(data string) {
		// 去掉 "data: " 前缀
		jsonStr := data
		if strings.HasPrefix(data, "data: ") {
			jsonStr = strings.TrimPrefix(data, "data: ")
		}
		jsonStr = strings.TrimSpace(jsonStr)

		if jsonStr == "" || jsonStr == "[DONE]" {
			return
		}

		var packet struct {
			P       string          `json:"p"`
			V       json.RawMessage `json:"v"`
			Choices []struct {
				Delta struct {
					Content string `json:"content"`
				} `json:"delta"`
			} `json:"choices"`
			Type    string          `json:"type"`
			Results json.RawMessage `json:"results"`
		}

		if err := json.Unmarshal([]byte(jsonStr), &packet); err != nil {
			// 静默忽略解析失败的数据
			return
		}

		citationMu.Lock()
		defer citationMu.Unlock()

		// 1. 原始 p/v 结构 (DeepSeek 经典) - 提取 citations
		if strings.Contains(packet.P, "results") {
			var results []struct {
				URL          string `json:"url"`
				Title        string `json:"title"`
				Snippet      string `json:"snippet"`
				QueryIndexes []int  `json:"query_indexes"`
			}
			if err := json.Unmarshal(packet.V, &results); err == nil && len(results) > 0 {
				newCount := 0
				for _, res := range results {
					cit := Citation{
						URL:          res.URL,
						Title:        res.Title,
						Snippet:      res.Snippet,
						QueryIndexes: res.QueryIndexes,
					}
					if u, err := url.Parse(res.URL); err == nil {
						cit.Domain = u.Host
					}
					exists := false
					for _, e := range capturedCitations {
						if e.URL == cit.URL {
							exists = true
							break
						}
					}
					if !exists && cit.URL != "" {
						capturedCitations = append(capturedCitations, cit)
						newCount++
					}
				}
				if newCount > 0 {
					d.logger.InfoWithContext(ctx, "[DEEPSEEK-SSE] Captured citations", map[string]interface{}{
						"new": newCount, "total": len(capturedCitations),
					}, nil)
				}
			}
		}

		// 2. 提取 Queries - 多种数据结构支持
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

		prevQueryCount := len(capturedQueries)

		// 2.1 完整 response/fragments 结构 (p="" 或 p="response")
		if packet.P == "" || packet.P == "response" {
			var vObj struct {
				Response struct {
					Fragments []struct {
						Type    string `json:"type"`
						Queries []struct {
							Query string `json:"query"`
							Text  string `json:"text"`
						} `json:"queries"`
					} `json:"fragments"`
				} `json:"response"`
			}
			if len(packet.V) > 0 {
				if err := json.Unmarshal(packet.V, &vObj); err == nil {
					for _, frag := range vObj.Response.Fragments {
						// 只处理 SEARCH 类型的 fragment
						if frag.Type == "SEARCH" || frag.Type == "" {
							for _, q := range frag.Queries {
								queryText := q.Query
								if queryText == "" {
									queryText = q.Text
								}
								addQuery(queryText)
							}
						}
					}
				}
			}
		}

		// 2.2 增量更新 queries (p 包含 "queries"，v 是数组)
		if strings.Contains(packet.P, "queries") {
			var queries []struct {
				Query string `json:"query"`
				Text  string `json:"text"`
			}
			if len(packet.V) > 0 {
				if err := json.Unmarshal(packet.V, &queries); err == nil {
					for _, q := range queries {
						queryText := q.Query
						if queryText == "" {
							queryText = q.Text
						}
						addQuery(queryText)
					}
				}
			}
		}

		// 2.3 顶层 queries 字段 (备用)
		if packet.P == "" || strings.Contains(packet.P, "query") {
			var vObj struct {
				Queries []struct {
					Query string `json:"query"`
					Text  string `json:"text"`
				} `json:"queries"`
			}
			if len(packet.V) > 0 {
				if err := json.Unmarshal(packet.V, &vObj); err == nil {
					for _, q := range vObj.Queries {
						queryText := q.Query
						if queryText == "" {
							queryText = q.Text
						}
						addQuery(queryText)
					}
				}
			}
		}

		// 只在有新 query 时打印
		if len(capturedQueries) > prevQueryCount {
			d.logger.InfoWithContext(ctx, "[DEEPSEEK-SSE] Captured queries", map[string]interface{}{
				"new":     len(capturedQueries) - prevQueryCount,
				"total":   len(capturedQueries),
				"queries": capturedQueries,
			}, nil)
		}

		// 3. 备用结构 - 根级 results
		if len(packet.Results) > 0 {
			var results []struct {
				URL   string `json:"url"`
				Title string `json:"title"`
			}
			if err := json.Unmarshal(packet.Results, &results); err == nil && len(results) > 0 {
				newCount := 0
				for _, res := range results {
					cit := Citation{URL: res.URL, Title: res.Title}
					if u, err := url.Parse(res.URL); err == nil {
						cit.Domain = u.Host
					}
					exists := false
					for _, e := range capturedCitations {
						if e.URL == cit.URL {
							exists = true
							break
						}
					}
					if !exists && cit.URL != "" {
						capturedCitations = append(capturedCitations, cit)
						newCount++
					}
				}
				if newCount > 0 {
					d.logger.InfoWithContext(ctx, "[DEEPSEEK-SSE] Captured citations (root)", map[string]interface{}{
						"new": newCount, "total": len(capturedCitations),
					}, nil)
				}
			}
		}
	}

	// 1. 启用 Network 域
	if err := (proto.NetworkEnable{}).Call(page); err != nil {
		d.logger.WarnWithContext(ctx, "[DEEPSEEK-RPA] Failed to enable network", map[string]interface{}{"error": err.Error()}, nil)
	}

	go func() {
		var mu sync.Mutex
		requestURLs := make(map[proto.NetworkRequestID]string)

		d.logger.InfoWithContext(ctx, "[DEEPSEEK-NET] Network listener started", nil, nil)

		page.EachEvent(func(e *proto.NetworkRequestWillBeSent) bool {
			select {
			case <-ctx.Done():
				return true
			default:
			}
			mu.Lock()
			requestURLs[e.RequestID] = e.Request.URL
			mu.Unlock()
			// 静默记录请求，不打印日志
			return false
		}, func(e *proto.NetworkRequestWillBeSentExtraInfo) bool {
			select {
			case <-ctx.Done():
				return true
			default:
			}
			// 静默处理，不打印日志
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
			// 静默记录响应，不打印日志
			return false
		}, func(e *proto.NetworkResponseReceivedExtraInfo) bool {
			select {
			case <-ctx.Done():
				return true
			default:
			}
			// 静默处理，不打印日志
			return false
		}, func(e *proto.NetworkDataReceived) bool {
			select {
			case <-ctx.Done():
				return true
			default:
			}
			// 静默处理，不打印日志（此事件非常频繁）
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
			// 静默处理，不获取响应体（减少开销）
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
			// 静默处理失败请求
			return false
		}, func(e *proto.NetworkWebSocketCreated) bool {
			select {
			case <-ctx.Done():
				return true
			default:
			}
			// 静默处理 WebSocket
			return false
		}, func(e *proto.NetworkWebSocketHandshakeResponseReceived) bool {
			select {
			case <-ctx.Done():
				return true
			default:
			}
			// 静默处理
			return false
		}, func(e *proto.NetworkWebSocketFrameReceived) bool {
			select {
			case <-ctx.Done():
				return true
			default:
			}
			// 静默处理 WebSocket 帧
			return false
		}, func(e *proto.NetworkWebSocketClosed) bool {
			select {
			case <-ctx.Done():
				return true
			default:
			}
			// 静默处理
			return false
		}, func(e *proto.NetworkEventSourceMessageReceived) bool {
			select {
			case <-ctx.Done():
				return true
			default:
			}

			// 直接处理 SSE 数据，不打印每条消息的详细日志
			parseSSEData(e.Data)

			return false
		})() // 调用 EachEvent 返回的 wait 函数以启动监听
	}()

	// 2. 强制绕过 Service Worker
	if err := (proto.NetworkSetBypassServiceWorker{Bypass: true}).Call(page); err != nil {
		d.logger.WarnWithContext(ctx, "[DEEPSEEK-RPA] Failed to set bypass service worker", map[string]interface{}{"error": err.Error()}, nil)
	}

	// 注入 Fetch 和 XHR 劫持脚本
	hijackScript := `(() => {
		console.log('__DS_DEBUG__: Hijack script injected v2');
		
		// 1. 劫持 Fetch
		const originalFetch = window.fetch;
		window.fetch = async (...args) => {
			const response = await originalFetch(...args);
			
			try {
				const urlStr = args[0] instanceof Request ? args[0].url : String(args[0]);
				console.log('__DS_REQ__:FETCH:' + urlStr);
				
				// 宽松匹配：包含 chat 或 api 的请求
				if (urlStr.includes('chat') || urlStr.includes('/api/')) {
					console.log('__DS_DEBUG__: Fetch matched, starting stream read');
					const clone = response.clone();
					const reader = clone.body.getReader();
					const decoder = new TextDecoder();
					
					(async () => {
						try {
							let buffer = '';
							while (true) {
								const { done, value } = await reader.read();
								if (done) {
									console.log('__DS_DEBUG__: Fetch stream done');
									break;
								}
								const chunk = decoder.decode(value, { stream: true });
								buffer += chunk;
								
								const lines = buffer.split('\n');
								buffer = lines.pop(); 
								
								for (const line of lines) {
									if (line.trim() === '') continue;
									console.log('__DS_LINE__:' + line);
								}
							}
						} catch (e) {
							console.error('__DS_ERROR__:FETCH:', e.message);
						}
					})();
				}
			} catch (err) {
				console.error('__DS_ERROR__:FETCH_OUTER:', err.message);
			}
			
			return response;
		};
		
		// 2. 劫持 XMLHttpRequest (备用)
		const originalXHROpen = XMLHttpRequest.prototype.open;
		const originalXHRSend = XMLHttpRequest.prototype.send;
		
		XMLHttpRequest.prototype.open = function(method, url, ...rest) {
			this._dsUrl = url;
			console.log('__DS_REQ__:XHR:' + url);
			return originalXHROpen.call(this, method, url, ...rest);
		};
		
		XMLHttpRequest.prototype.send = function(...args) {
			if (this._dsUrl && (this._dsUrl.includes('chat') || this._dsUrl.includes('/api/'))) {
				console.log('__DS_DEBUG__: XHR matched, attaching listener');
				this.addEventListener('readystatechange', () => {
					if (this.readyState === 3 || this.readyState === 4) {
						try {
							const text = this.responseText;
							if (text) {
								const lines = text.split('\n');
								for (const line of lines) {
									if (line.trim() && line.startsWith('data:')) {
										console.log('__DS_LINE__:' + line);
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
				console.log('__DS_REQ__:SSE:' + url);
				const es = new originalEventSource(url, config);
				es.addEventListener('message', (e) => {
					console.log('__DS_LINE__:data: ' + e.data);
				});
				return es;
			};
		}
		
		console.log('__DS_DEBUG__: All hijacks installed');
	})()`

	if _, err := page.EvalOnNewDocument(hijackScript); err != nil {
		d.logger.WarnWithContext(ctx, "[DEEPSEEK-RPA] Failed to inject fetch hijacker", map[string]interface{}{"error": err.Error()}, nil)
	}

	// 2. 监听 Console 事件处理数据
	go func() {
		// 启用 Runtime 域以接收 console 事件
		if err := (proto.RuntimeEnable{}).Call(page); err != nil {
			d.logger.WarnWithContext(ctx, "[DEEPSEEK-SSE] RuntimeEnable failed", map[string]interface{}{"error": err.Error()}, nil)
		}

		page.EachEvent(func(e *proto.RuntimeConsoleAPICalled) bool {
			select {
			case <-ctx.Done():
				return true
			default:
			}

			// 处理 console.log 与 console.error（__DS_ERROR__ 通过 console.error 打出）
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
				if strings.HasPrefix(logMsg, "__DS_LINE__:") {
					line := strings.TrimPrefix(logMsg, "__DS_LINE__:")
					line = strings.TrimSpace(line)
					// 直接解析，不打印每条日志
					parseSSEData(line)
				} else if strings.HasPrefix(logMsg, "__DS_ERROR__:") {
					// 只打印错误
					d.logger.WarnWithContext(ctx, "[DEEPSEEK-JS] ERR: "+strings.TrimPrefix(logMsg, "__DS_ERROR__:"), nil, nil)
				}
				// 忽略 __DS_DEBUG__ 和 __DS_REQ__ 日志
			}
			return false
		})() // 调用 EachEvent 返回的 wait 函数以启动监听
	}()

	page.MustNavigate(homeURL)
	page.MustWaitLoad()

	d.logger.InfoWithContext(ctx, "[DEEPSEEK-RPA] Waiting for page stability (3s)...", nil, nil)
	rod.Try(func() {
		page.Timeout(3 * time.Second).WaitStable(1 * time.Second)
	})

	d.logPageSnapshot(ctx, page, "AFTER_LOAD")

	d.logger.InfoWithContext(ctx, "[DEEPSEEK-RPA] ===== STEP 1: Enable Web Search =====", nil, nil)
	webSearchEnabled := d.tryEnableWebSearch(ctx, page)
	d.logger.InfoWithContext(ctx, "[DEEPSEEK-RPA] Web search toggle result", map[string]interface{}{"enabled": webSearchEnabled}, nil)

	d.logger.InfoWithContext(ctx, "[DEEPSEEK-RPA] ===== STEP 2: Find Input Textarea =====", nil, nil)
	textarea, err := d.findTextarea(ctx, page)
	if err != nil {
		return nil, err
	}

	d.logger.InfoWithContext(ctx, "[DEEPSEEK-RPA] ===== STEP 3: Input Keyword =====", nil, nil)
	err = d.inputKeyword(ctx, page, textarea, keyword)
	if err != nil {
		return nil, err
	}

	d.logger.InfoWithContext(ctx, "[DEEPSEEK-RPA] ===== STEP 4: Find Submit Button =====", nil, nil)
	submitBtn, err := d.findSubmitButton(ctx, page)
	if err != nil {
		return nil, err
	}

	d.logger.InfoWithContext(ctx, "[DEEPSEEK-RPA] ===== STEP 5: Click Submit =====", nil, nil)
	err = d.clickSubmit(ctx, submitBtn, page)
	if err != nil {
		return nil, err
	}

	d.logger.InfoWithContext(ctx, "[DEEPSEEK-RPA] ===== STEP 6: Wait for Response (via polling) =====", nil, nil)
	d.logger.InfoWithContext(ctx, "[DEEPSEEK-RPA] Waiting for AI response...", nil, nil)
	time.Sleep(5 * time.Second)

	d.waitForResponseComplete(ctx, page)

	d.logger.InfoWithContext(ctx, "[DEEPSEEK-RPA] Proceeding to final extraction", nil, nil)
	d.logPageSnapshot(ctx, page, "AFTER_RESPONSE")

	d.logger.InfoWithContext(ctx, "[DEEPSEEK-RPA] ===== STEP 7: Extract Response =====", nil, nil)
	fullText, err := d.extractResponse(ctx, page)
	if err != nil {
		return nil, err
	}

	d.logger.InfoWithContext(ctx, "[DEEPSEEK-RPA] ========== SEARCH COMPLETE ==========", map[string]interface{}{
		"response_length":  len(fullText),
		"response_preview": truncateString(fullText, 200),
	}, nil)

	citationMu.Lock()
	finalCitations := make([]Citation, len(capturedCitations))
	copy(finalCitations, capturedCitations)
	finalQueries := make([]string, len(capturedQueries))
	copy(finalQueries, capturedQueries)
	citationMu.Unlock()

	// 不使用用户搜索词兜底，如果没有捕获到查询则返回空数组

	// 根据 QueryIndexes[0] 填充每个 Citation 的 Query 字段
	for i := range finalCitations {
		if len(finalCitations[i].QueryIndexes) > 0 {
			idx := finalCitations[i].QueryIndexes[0]
			if idx >= 0 && idx < len(finalQueries) {
				finalCitations[i].Query = finalQueries[idx]
			}
		}
	}

	// DOM 兜底策略
	// if len(finalCitations) == 0 {
	// 	d.logger.WarnWithContext(ctx, "[DEEPSEEK-RPA] JS injection missed citations, trying DOM fallback...", nil, nil)
	// 	domCitations := d.extractCitationsFromDOM(ctx, page)
	// 	if len(domCitations) > 0 {
	// 		finalCitations = domCitations
	// 		d.logger.InfoWithContext(ctx, "[DEEPSEEK-RPA] DOM fallback success", map[string]interface{}{"count": len(finalCitations)}, nil)
	// 	} else {
	// 		d.logger.WarnWithContext(ctx, "[DEEPSEEK-RPA] DOM fallback also failed", nil, nil)
	// 	}
	// }

	return &SearchResult{
		Queries:   finalQueries,
		Citations: finalCitations,
		FullText:  fullText,
	}, nil
}

// 增强版 checkToggleActive (全面检查元素及其上下文的激活状态)
func (d *DeepSeekProvider) checkToggleActive(ctx context.Context, elem *rod.Element) (bool, error) {
	// JS 注入检查：遍历元素、父级、子级、兄弟及其上下文，检查 class 名和颜色
	isActive, err := elem.Eval(`() => {
		const el = this;
		
		// 收集检查目标：自身、祖先（3级）、以及这些祖先的所有子孙
		// 这样可以覆盖同一容器内的所有相关状态元素
		const targets = new Set();
		let current = el;
		for (let i = 0; i < 3 && current; i++) {
			targets.add(current);
			// 包含祖先的所有子元素 (通常包含同级的 switch 按钮)
			current.querySelectorAll('*').forEach(t => targets.add(t));
			current = current.parentElement;
		}
		
		const activeKeywords = ['active', 'checked', 'selected', 'enabled', 'on', '--checked', '-active'];
		// 蓝色系/紫色系颜色（DeepSeek 激活色）
		const activeColors = [
			'rgb(77, 107, 254)', 
			'rgb(36, 127, 255)', 
			'#4d6bfe',
			'rgb(24, 144, 255)', 
			'rgb(0, 82, 204)',
			'rgb(103, 58, 183)', // DeepThink Purple
			'rgb(77, 107, 254)'  // DeepSeek Blue
		]; 

		for (const t of targets) {
			// 1. 检查 Class (最可靠)
			if (t.className && typeof t.className === 'string') {
				const cls = t.className.toLowerCase();
				if (activeKeywords.some(k => cls.includes(k))) return true;
			}
			
			// 2. 检查属性
			if (t.getAttribute('aria-checked') === 'true') return true;
			if (t.getAttribute('data-state') === 'checked') return true;
			if (t.getAttribute('data-active') === 'true') return true;
			if (t.tagName === 'INPUT' && (t.type === 'checkbox' || t.type === 'radio') && t.checked) return true;

			// 3. 检查颜色 (Color, Bg, Fill, Stroke)
			const style = window.getComputedStyle(t);
			const props = ['color', 'backgroundColor', 'fill', 'stroke', 'borderColor'];
			
			for (const p of props) {
				const val = style[p];
				if (val && val !== 'none' && val !== 'transparent' && val !== 'rgba(0, 0, 0, 0)') {
					// 检查是否在预定义的激活色列表中
					if (activeColors.some(c => val.includes(c))) return true;
					
					// 泛化检查：如果是蓝色系（R < G < B 且 B 较大）
					if (val.startsWith('rgb')) {
						const parts = val.match(/\d+/g);
						if (parts && parts.length >= 3) {
							const r = parseInt(parts[0]);
							const g = parseInt(parts[1]);
							const b = parseInt(parts[2]);
							// 典型的 DeepSeek 蓝色是 (77, 107, 254)
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

// 增强版 clickToggleIfInactive
func (d *DeepSeekProvider) clickToggleIfInactive(ctx context.Context, elem *rod.Element, selector string) bool {
	isActivated, err := d.checkToggleActive(ctx, elem)
	if err != nil {
		d.logger.WarnWithContext(ctx, "[DEEPSEEK-RPA] State check error", map[string]interface{}{"error": err.Error()}, nil)
	}

	if isActivated {
		d.logger.InfoWithContext(ctx, "[DEEPSEEK-RPA] Toggle ALREADY ACTIVE, skipping click", map[string]interface{}{
			"selector": selector,
		}, nil)
		return true
	}

	d.logger.InfoWithContext(ctx, "[DEEPSEEK-RPA] Toggle inactive (or state undetected), performing click...", map[string]interface{}{
		"selector": selector,
	}, nil)

	clickErr := rod.Try(func() {
		elem.ScrollIntoView()
		elem.MustClick()
	})

	if clickErr == nil {
		// 等待状态变化
		time.Sleep(1 * time.Second)

		// 再次检查确认是否成功开启
		isActivatedNow, _ := d.checkToggleActive(ctx, elem)
		if isActivatedNow {
			d.logger.InfoWithContext(ctx, "[DEEPSEEK-RPA] Toggle successfully activated", nil, nil)
			return true
		} else {
			d.logger.WarnWithContext(ctx, "[DEEPSEEK-RPA] Toggle clicked but state check returned false (might be false negative)", nil, nil)
			// 即使检查返回 false，我们也认为已经尽力了，避免死循环点击
			return true
		}
	}

	d.logger.WarnWithContext(ctx, "[DEEPSEEK-RPA] Toggle click failed", map[string]interface{}{
		"selector": selector,
		"error":    clickErr.Error(),
	}, nil)
	return false
}

// 增强版 tryEnableWebSearch
func (d *DeepSeekProvider) tryEnableWebSearch(ctx context.Context, page *rod.Page) bool {
	// 1. 文本精准匹配
	textPatterns := []string{"联网搜索", "Search the web", "Web Search"}

	for _, pattern := range textPatterns {
		xpath := fmt.Sprintf("//*[self::div or self::button or self::span or self::label][contains(text(), '%s')]", pattern)
		elems, err := page.ElementsX(xpath)

		if err == nil {
			for _, elem := range elems {
				if visible, _ := elem.Visible(); visible {
					d.logger.InfoWithContext(ctx, "[DEEPSEEK-RPA] Found toggle by text", map[string]interface{}{
						"pattern": pattern,
					}, nil)
					// 文本匹配到的通常是 Label 或按钮本身，可信度高
					if d.clickToggleIfInactive(ctx, elem, "text:"+pattern) {
						return true
					}
				}
			}
		}
	}

	// 2. 选择器匹配 (更精确的类名)
	selectors := []string{
		// 精确 Class
		".ds-toggle-button",
		".ds-switch",
		// 属性
		"div[class*='search'] input[type='checkbox']",
		"[aria-label*='联网']",
		"[title*='联网']",
	}

	for _, sel := range selectors {
		elems, err := page.Timeout(2 * time.Second).Elements(sel)
		if err == nil {
			for _, elem := range elems {
				if visible, _ := elem.Visible(); visible {
					d.logger.InfoWithContext(ctx, "[DEEPSEEK-RPA] Found potential toggle by selector", map[string]interface{}{
						"selector": sel,
					}, nil)

					if d.clickToggleIfInactive(ctx, elem, sel) {
						return true
					}
				}
			}
		}
	}

	d.logger.WarnWithContext(ctx, "[DEEPSEEK-RPA] Could not enable web search toggle (no match found)", nil, nil)
	return false
}

// extractCitationsFromDOM attempts to find citations in the rendered page
func (d *DeepSeekProvider) extractCitationsFromDOM(ctx context.Context, page *rod.Page) []Citation {
	var citations []Citation

	// 通用逻辑：扫描所有外部链接
	elements, err := page.Elements("a[href^='http']")
	if err != nil {
		return citations
	}

	noiseDomains := []string{
		"deepseek.com",
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

		// 过滤噪音关键词（标题或链接）
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
		if class != nil && (strings.Contains(*class, "search") || strings.Contains(*class, "result") || strings.Contains(*class, "source") || strings.Contains(*class, "reference")) {
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

func (d *DeepSeekProvider) logPageSnapshot(ctx context.Context, page *rod.Page, phase string) {
	d.logger.InfoWithContext(ctx, fmt.Sprintf("[DEEPSEEK-RPA] [%s] Taking page snapshot...", phase), nil, nil)

	url := page.MustInfo().URL
	title := page.MustInfo().Title
	d.logger.InfoWithContext(ctx, fmt.Sprintf("[DEEPSEEK-RPA] [%s] Page info", phase), map[string]interface{}{
		"url":   url,
		"title": title,
	}, nil)

	selectors := []string{
		"textarea",
		"textarea#chat-input",
		"div[class*='sendButton']",
		"button[class*='send']",
		"div[class*='ds-markdown']",
		"div[class*='search']",
		"div[class*='toggle']",
		"input[type='checkbox']",
		"span[class*='switch']",
	}

	for _, sel := range selectors {
		elements, err := page.Elements(sel)
		count := 0
		if err == nil {
			count = len(elements)
		}
		d.logger.InfoWithContext(ctx, fmt.Sprintf("[DEEPSEEK-RPA] [%s] Selector check", phase), map[string]interface{}{
			"selector": sel,
			"found":    count,
		}, nil)
	}

	textPatterns := []string{"联网", "搜索", "Search", "深度思考", "DeepThink", "Send", "发送"}
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
				info += " class=" + truncateString(*class, 50)
			}
		}
		d.logger.InfoWithContext(ctx, fmt.Sprintf("[DEEPSEEK-RPA] [%s] Text pattern check", phase), map[string]interface{}{
			"pattern": pattern,
			"found":   has,
			"element": info,
		}, nil)
	}
}

// ... rest of file (findTextarea, etc)
func (d *DeepSeekProvider) findTextarea(ctx context.Context, page *rod.Page) (*rod.Element, error) {
	selectors := []struct {
		selector string
		desc     string
	}{
		{"textarea#chat-input", "id=chat-input"},
		{"textarea[placeholder*='Message']", "placeholder contains Message"},
		{"textarea[placeholder*='message']", "placeholder contains message"},
		{"textarea[placeholder*='输入']", "placeholder contains 输入"},
		{"textarea[placeholder*='发送']", "placeholder contains 发送"},
		{"textarea[class*='input']", "class contains input"},
		{"textarea", "any textarea"},
	}

	for _, s := range selectors {
		d.logger.InfoWithContext(ctx, "[DEEPSEEK-RPA] Trying textarea selector", map[string]interface{}{
			"selector": s.selector,
			"desc":     s.desc,
		}, nil)

		elem, err := page.Timeout(3 * time.Second).Element(s.selector)
		if err == nil && elem != nil {
			placeholder, _ := elem.Attribute("placeholder")
			class, _ := elem.Attribute("class")
			id, _ := elem.Attribute("id")

			d.logger.InfoWithContext(ctx, "[DEEPSEEK-RPA] Textarea found!", map[string]interface{}{
				"selector":    s.selector,
				"placeholder": safeString(placeholder),
				"class":       truncateString(safeString(class), 50),
				"id":          safeString(id),
			}, nil)
			return elem, nil
		}
		d.logger.InfoWithContext(ctx, "[DEEPSEEK-RPA] Textarea selector failed", map[string]interface{}{
			"selector": s.selector,
		}, nil)
	}

	return nil, fmt.Errorf("no textarea found with any selector")
}

func (d *DeepSeekProvider) inputKeyword(ctx context.Context, page *rod.Page, textarea *rod.Element, keyword string) error {
	d.logger.InfoWithContext(ctx, "[DEEPSEEK-RPA] Clicking textarea...", nil, nil)
	clickErr := rod.Try(func() {
		textarea.MustClick()
	})
	if clickErr != nil {
		d.logger.WarnWithContext(ctx, "[DEEPSEEK-RPA] Click failed", map[string]interface{}{"error": clickErr.Error()}, nil)
	}

	d.logger.InfoWithContext(ctx, "[DEEPSEEK-RPA] Clearing existing text (Ctrl+A, Backspace)...", nil, nil)
	page.KeyActions().Press(input.ControlLeft).Type('a').Release(input.ControlLeft).Press(input.Backspace).MustDo()

	d.logger.InfoWithContext(ctx, "[DEEPSEEK-RPA] Inputting keyword...", map[string]interface{}{"keyword": keyword}, nil)

	textarea.MustClick()

	if err := textarea.SelectAllText(); err != nil {
		d.logger.WarnWithContext(ctx, "[DEEPSEEK-RPA] SelectAllText failed", map[string]interface{}{"error": err.Error()}, nil)
	}

	if err := textarea.Input(keyword); err != nil {
		d.logger.WarnWithContext(ctx, "[DEEPSEEK-RPA] Input failed, trying MustInput", map[string]interface{}{"error": err.Error()}, nil)
		textarea.MustInput(keyword)
	}

	time.Sleep(500 * time.Millisecond)

	value, _ := textarea.Text()
	d.logger.InfoWithContext(ctx, "[DEEPSEEK-RPA] Textarea value after input", map[string]interface{}{
		"expected": keyword,
		"actual":   value,
		"match":    value == keyword,
	}, nil)

	return nil
}

func (d *DeepSeekProvider) findSubmitButton(ctx context.Context, page *rod.Page) (*rod.Element, error) {
	selectors := []struct {
		selector string
		desc     string
	}{
		{`[data-testid="chat-input-send-button"]`, "data-testid send button"},
		{"#chat-input-send-button", "id send button"},
		{"div[class*='_7436']", "class _7436 (from HTML sample)"},
		{"div[role='button'] .ds-icon", "div role button with icon"},
		{".ds-icon-button", "class ds-icon-button"},
		{"div[role='button']:has(svg)", "div role button with svg"},
		{"button:has(svg)", "button with svg"},
	}

	for _, s := range selectors {
		d.logger.InfoWithContext(ctx, "[DEEPSEEK-RPA] Trying submit selector", map[string]interface{}{
			"selector": s.selector,
			"desc":     s.desc,
		}, nil)

		elem, err := page.Timeout(2 * time.Second).Element(s.selector)
		if err == nil && elem != nil {
			disabled, _ := elem.Attribute("disabled")
			class, _ := elem.Attribute("class")

			d.logger.InfoWithContext(ctx, "[DEEPSEEK-RPA] Submit button found!", map[string]interface{}{
				"selector": s.selector,
				"disabled": disabled != nil,
				"class":    truncateString(safeString(class), 50),
			}, nil)
			return elem, nil
		}
		d.logger.InfoWithContext(ctx, "[DEEPSEEK-RPA] Submit selector failed", map[string]interface{}{
			"selector": s.selector,
		}, nil)
	}

	d.logger.InfoWithContext(ctx, "[DEEPSEEK-RPA] Trying text-based submit search...", nil, nil)
	textPatterns := []string{"发送", "Send"}
	for _, pattern := range textPatterns {
		elem, err := page.Timeout(2*time.Second).ElementR("div, button", pattern)
		if err == nil && elem != nil {
			d.logger.InfoWithContext(ctx, "[DEEPSEEK-RPA] Found submit by text", map[string]interface{}{
				"pattern": pattern,
			}, nil)
			return elem, nil
		}
	}

	return nil, fmt.Errorf("no submit button found with any selector")
}

func (d *DeepSeekProvider) clickSubmit(ctx context.Context, submitBtn *rod.Element, page *rod.Page) error {
	disabled, _ := submitBtn.Attribute("disabled")
	if disabled != nil {
		d.logger.WarnWithContext(ctx, "[DEEPSEEK-RPA] Submit button is disabled!", nil, nil)
	} else {
		d.logger.InfoWithContext(ctx, "[DEEPSEEK-RPA] Clicking submit button...", nil, nil)
		err := submitBtn.Click(proto.InputMouseButtonLeft, 1)
		if err != nil {
			d.logger.WarnWithContext(ctx, "[DEEPSEEK-RPA] Click failed, trying MustClick", map[string]interface{}{"error": err.Error()}, nil)
			submitBtn.MustClick()
		}

		time.Sleep(1 * time.Second)
	}

	d.logger.InfoWithContext(ctx, "[DEEPSEEK-RPA] Sending ENTER key as backup...", nil, nil)
	page.KeyActions().Press(input.Enter).MustDo()

	d.logger.InfoWithContext(ctx, "[DEEPSEEK-RPA] Submit action completed", nil, nil)
	return nil
}

func (d *DeepSeekProvider) waitForResponse(ctx context.Context, page *rod.Page) error {
	d.logger.InfoWithContext(ctx, "[DEEPSEEK-RPA] Waiting for page idle...", nil, nil)
	page.MustWaitIdle()

	d.logger.InfoWithContext(ctx, "[DEEPSEEK-RPA] Waiting for response elements (30s timeout)...", nil, nil)

	responseSelectors := []string{
		"div[class*='ds-markdown']",
		"div[class*='message-content']",
		"div[class*='assistant']",
		"div[class*='response']",
	}

	foundResponse := false
	for _, sel := range responseSelectors {
		d.logger.InfoWithContext(ctx, "[DEEPSEEK-RPA] Checking for response selector", map[string]interface{}{
			"selector": sel,
		}, nil)

		err := rod.Try(func() {
			page.Timeout(10*time.Second).MustWaitElementsMoreThan(sel, 0)
		})
		if err == nil {
			d.logger.InfoWithContext(ctx, "[DEEPSEEK-RPA] Response elements found!", map[string]interface{}{
				"selector": sel,
			}, nil)
			foundResponse = true
			break
		}
	}

	if !foundResponse {
		d.logger.WarnWithContext(ctx, "[DEEPSEEK-RPA] No response elements found with standard selectors", nil, nil)
	}

	d.logger.InfoWithContext(ctx, "[DEEPSEEK-RPA] Waiting for page idle again...", nil, nil)
	page.MustWaitIdle()

	d.logger.InfoWithContext(ctx, "[DEEPSEEK-RPA] Additional 3s wait for streaming to complete...", nil, nil)
	time.Sleep(3 * time.Second)

	return nil
}

func (d *DeepSeekProvider) waitForResponseComplete(ctx context.Context, page *rod.Page) {
	const maxRetries = 30
	lastContent := ""

	for i := 0; i < maxRetries; i++ {
		time.Sleep(2 * time.Second)

		elements, err := page.Elements(".ds-markdown")
		if err == nil && len(elements) > 0 {
			lastElem := elements[len(elements)-1]
			currentContent := lastElem.MustText()

			if len(currentContent) > 100 && currentContent == lastContent {
				// 内容不再变化，检查是否有停止按钮
				hasStopBtn, _, _ := page.HasR("*", "停止生成")
				if !hasStopBtn {
					d.logger.InfoWithContext(ctx, "[DEEPSEEK-RPA] Response generation complete", nil, nil)
					return
				}
			}

			lastContent = currentContent
		}
		// 如果元素查找失败，继续下一次循环
	}
}

func (d *DeepSeekProvider) extractResponse(ctx context.Context, page *rod.Page) (string, error) {
	responseSelectors := []struct {
		selector string
		desc     string
	}{
		{"div[class*='ds-markdown']", "ds-markdown"},
		{"div[class*='message-content']", "message-content"},
		{"div[class*='assistant-message']", "assistant-message"},
		{"div[class*='response']", "response"},
		{"div[class*='markdown']", "markdown"},
	}

	for _, s := range responseSelectors {
		d.logger.InfoWithContext(ctx, "[DEEPSEEK-RPA] Trying response selector", map[string]interface{}{
			"selector": s.selector,
			"desc":     s.desc,
		}, nil)

		elements, err := page.Elements(s.selector)
		if err == nil && len(elements) > 0 {
			lastElem := elements[len(elements)-1]
			text := lastElem.MustText()

			d.logger.InfoWithContext(ctx, "[DEEPSEEK-RPA] Response extracted!", map[string]interface{}{
				"selector":      s.selector,
				"element_count": len(elements),
				"text_length":   len(text),
				"preview":       truncateString(text, 100),
			}, nil)

			if len(strings.TrimSpace(text)) > 0 {
				return text, nil
			}
		}
	}

	d.logger.WarnWithContext(ctx, "[DEEPSEEK-RPA] Falling back to body text", nil, nil)
	bodyText := page.MustElement("body").MustText()
	d.logger.InfoWithContext(ctx, "[DEEPSEEK-RPA] Body text extracted", map[string]interface{}{
		"length": len(bodyText),
	}, nil)

	return bodyText, nil
}

func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

func safeString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
