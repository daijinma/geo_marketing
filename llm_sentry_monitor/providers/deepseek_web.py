import time
import os
import json
import re
from playwright.sync_api import sync_playwright
from providers.base import BaseProvider
from core.parser import extract_domain
from core.logger_config import setup_logger

class DeepSeekWebProvider(BaseProvider):
    def search(self, keyword: str, prompt: str):
        user_data_dir = os.path.join(os.getenv("BROWSER_DATA_DIR", "./browser_data"), "deepseek")
        
        # 用于存储拦截到的搜索结果
        captured_search_results = []
        captured_queries = []  # 存储 AI 拓展的搜索词
        full_response_text = ""
        
        def handle_response(response):
            """拦截 API 响应，提取搜索结果和拓展词"""
            nonlocal captured_search_results, captured_queries, full_response_text
            
            url_lower = response.url.lower()
            # 扩展API端点匹配模式
            api_patterns = [
                "api/v0/chat/completion",
                "api/v1/chat/completion"
            ]
            self.logger.info(f"[网络拦截] 响应URL: {response.url}")
            if any(pattern in url_lower for pattern in api_patterns):
                matched_pattern = next((p for p in api_patterns if p in url_lower), "unknown")
                self.logger.info(f"[网络拦截] API端点匹配: {matched_pattern}")
                try:
                    content_type = response.headers.get("content-type", "")
                    
                    # 处理 SSE 流
                    if "text/event-stream" in content_type or "stream" in url_lower:
                        try:
                            body = response.text()
                            self.logger.info(f"[网络拦截] SSE流式响应，开始解析数据")
                            
                            # 正确解析 SSE 数据流
                            # SSE 格式：事件之间用空行分隔，一个事件可以有多行 data:
                            events = []
                            current_event_data = []
                            
                            for line in body.split('\n'):
                                line = line.rstrip('\r')  # 移除可能的 \r
                                
                                if line.startswith('data: '):
                                    # 收集多行 data: 字段
                                    data_content = line[6:]  # 去掉 "data: " 前缀
                                    current_event_data.append(data_content)
                                elif line == '':
                                    # 空行表示事件结束，合并所有 data: 行
                                    if current_event_data:
                                        # 多行 data: 应该用换行符连接
                                        combined_data = '\n'.join(current_event_data)
                                        events.append(combined_data)
                                        current_event_data = []
                                elif line.startswith('event:') or line.startswith('id:') or line.startswith('retry:'):
                                    # 忽略其他 SSE 字段（event, id, retry）
                                    continue
                            
                            # 处理最后一个事件（如果没有以空行结尾）
                            if current_event_data:
                                combined_data = '\n'.join(current_event_data)
                                events.append(combined_data)
                            
                            self.logger.debug(f"[SSE解析] 共解析到 {len(events)} 个 SSE 事件")
                            
                            # 处理每个事件的数据
                            for event_data in events:
                                try:
                                    json_str = event_data.strip()
                                    if json_str and json_str != '[DONE]' and json_str != 'null':
                                        data = json.loads(json_str)
                                        
                                        # 提取搜索结果和拓展词
                                        if 'v' in data:
                                            # 情况1: 完整的 fragments 数据
                                            if isinstance(data['v'], dict):
                                                response_data = data['v'].get('response', {})
                                                fragments = response_data.get('fragments', [])
                                                for frag in fragments:
                                                    if frag.get('type') == 'SEARCH':
                                                        # 提取拓展词 (queries)
                                                        queries = frag.get('queries', [])
                                                        queries_before = len(captured_queries)
                                                        for q in queries:
                                                            if isinstance(q, dict):
                                                                query_text = q.get('query', q.get('text', ''))
                                                            else:
                                                                query_text = str(q)
                                                            if query_text and query_text not in captured_queries:
                                                                captured_queries.append(query_text)
                                                                self.logger.info(f"[数据抓取] 查询词: {query_text}")
                                                        
                                                        if len(captured_queries) > queries_before:
                                                            self.logger.info(f"[数据抓取] 进度: {len(captured_queries)} 个查询, {len(captured_search_results)} 个网站")
                                                        
                                                        # 提取搜索结果 (results)
                                                        results = frag.get('results', [])
                                                        results_before = len(captured_search_results)
                                                        for r in results:
                                                            if isinstance(r, dict) and r.get('url'):
                                                                url = r.get('url', '')
                                                                domain = extract_domain(url)
                                                                captured_search_results.append({
                                                                    "url": url,
                                                                    "title": r.get('title', r.get('name', '')),
                                                                    "snippet": r.get('snippet', r.get('description', '')),
                                                                    "site_name": r.get('site_name', r.get('source', '')),
                                                                    "cite_index": r.get('cite_index', r.get('index', 0))
                                                                })
                                                                self.logger.info(f"[数据抓取] 网站: {url[:60]}... (域名: {domain})")
                                                        
                                                        if len(captured_search_results) > results_before:
                                                            self.logger.info(f"[数据抓取] 进度: {len(captured_queries)} 个查询, {len(captured_search_results)} 个网站")
                                            
                                            # 情况2: 增量更新的 results 数组（关键修复）
                                            elif isinstance(data['v'], list):
                                                # 检查路径参数，确认是否是 results 更新
                                                path = data.get('p', '')
                                                
                                                # 处理增量更新的 results: {"p":"response/fragments/-1/results","v":[...]}
                                                if 'results' in path.lower() or (len(data['v']) > 0 and isinstance(data['v'][0], dict) and 'url' in data['v'][0]):
                                                    results_before = len(captured_search_results)
                                                    for r in data['v']:
                                                        if isinstance(r, dict) and r.get('url'):
                                                            url = r.get('url', '')
                                                            domain = extract_domain(url)
                                                            captured_search_results.append({
                                                                "url": url,
                                                                "title": r.get('title', r.get('name', '')),
                                                                "snippet": r.get('snippet', r.get('description', '')),
                                                                "site_name": r.get('site_name', r.get('source', '')),
                                                                "cite_index": r.get('cite_index', r.get('index', 0))
                                                            })
                                                            self.logger.info(f"从 API 增量更新捕获网站: {url[:60]}... (域名: {domain}, cite_index: {r.get('cite_index', 0)})")
                                                    
                                                    if len(captured_search_results) > results_before:
                                                        self.logger.info(f"当前已捕获: {len(captured_queries)} 个查询, {len(captured_search_results)} 个网站")
                                                
                                                # 处理增量更新的 queries: {"p":"response/fragments/-1/queries","v":[...]}
                                                elif 'queries' in path.lower() or (len(data['v']) > 0 and not isinstance(data['v'][0], dict)):
                                                    queries_before = len(captured_queries)
                                                    for q in data['v']:
                                                        if isinstance(q, dict):
                                                            query_text = q.get('query', q.get('text', ''))
                                                        else:
                                                            query_text = str(q)
                                                        if query_text and query_text not in captured_queries:
                                                            captured_queries.append(query_text)
                                                            self.logger.info(f"从 API 增量更新捕获查询: \"{query_text}\"")
                                                    
                                                    if len(captured_queries) > queries_before:
                                                        self.logger.info(f"当前已捕获: {len(captured_queries)} 个查询, {len(captured_search_results)} 个网站")
                                        
                                        # 尝试其他可能的数据结构
                                        # 直接包含 results 或 queries
                                        if 'results' in data and isinstance(data['results'], list):
                                            results_before = len(captured_search_results)
                                            for r in data['results']:
                                                if isinstance(r, dict) and r.get('url'):
                                                    url = r.get('url', '')
                                                    domain = extract_domain(url)
                                                    captured_search_results.append({
                                                        "url": url,
                                                        "title": r.get('title', r.get('name', '')),
                                                        "snippet": r.get('snippet', r.get('description', '')),
                                                        "site_name": r.get('site_name', r.get('source', '')),
                                                        "cite_index": r.get('cite_index', r.get('index', 0))
                                                    })
                                                    self.logger.info(f"从 SSE (results字段) 提取到网站: {url[:60]}... (域名: {domain})")
                                            
                                            if len(captured_search_results) > results_before:
                                                self.logger.info(f"当前已捕获: {len(captured_queries)} 个查询, {len(captured_search_results)} 个网站")
                                        
                                        if 'queries' in data and isinstance(data['queries'], list):
                                            queries_before = len(captured_queries)
                                            for q in data['queries']:
                                                if isinstance(q, dict):
                                                    query_text = q.get('query', q.get('text', ''))
                                                else:
                                                    query_text = str(q)
                                                if query_text and query_text not in captured_queries:
                                                    captured_queries.append(query_text)
                                                    self.logger.info(f"从 SSE (queries字段) 提取到查询: \"{query_text}\"")
                                            
                                            if len(captured_queries) > queries_before:
                                                self.logger.info(f"当前已捕获: {len(captured_queries)} 个查询, {len(captured_search_results)} 个网站")
                                        
                                        # 提取回答内容
                                        if 'content' in data:
                                            content = data.get('content', '')
                                            if isinstance(content, str) and content:
                                                full_response_text += content
                                        elif 'delta' in data and 'content' in data.get('delta', {}):
                                            content = data['delta'].get('content', '')
                                            if isinstance(content, str) and content:
                                                full_response_text += content
                                                
                                except json.JSONDecodeError as e:
                                    self.logger.debug(f"JSON 解析失败: {e}")
                                    continue
                        except Exception as e:
                            self.logger.debug(f"解析 SSE 响应失败: {e}")
                    
                    # 处理普通 JSON 响应
                    elif "application/json" in content_type:
                        try:
                            data = response.json()
                            self.logger.debug(f"拦截到 JSON 响应: {response.url[:100]}")
                            
                            # 提取搜索相关信息
                            if 'search' in data:
                                search_data = data['search']
                                if 'queries' in search_data:
                                    queries = search_data['queries']
                                    queries_before = len(captured_queries)
                                    if isinstance(queries, list):
                                        for q in queries:
                                            query_text = q if isinstance(q, str) else q.get('query', '')
                                            if query_text and query_text not in captured_queries:
                                                captured_queries.append(query_text)
                                                self.logger.info(f"从 JSON 响应提取到查询: \"{query_text}\"")
                                    
                                    if len(captured_queries) > queries_before:
                                        self.logger.info(f"当前已捕获: {len(captured_queries)} 个查询, {len(captured_search_results)} 个网站")
                                
                                if 'results' in search_data:
                                    results_before = len(captured_search_results)
                                    for r in search_data['results']:
                                        if isinstance(r, dict) and r.get('url'):
                                            url = r.get('url', '')
                                            domain = extract_domain(url)
                                            captured_search_results.append({
                                                "url": url,
                                                "title": r.get('title', ''),
                                                "snippet": r.get('snippet', ''),
                                                "site_name": r.get('site_name', r.get('source', '')),
                                                "cite_index": r.get('cite_index', r.get('index', 0))
                                            })
                                            self.logger.info(f"从 JSON 响应提取到网站: {url[:60]}... (域名: {domain})")
                                    
                                    if len(captured_search_results) > results_before:
                                        self.logger.info(f"当前已捕获: {len(captured_queries)} 个查询, {len(captured_search_results)} 个网站")
                        except Exception as e:
                            self.logger.debug(f"解析 JSON 响应失败: {e}")
                            
                except Exception as e:
                    self.logger.debug(f"拦截响应失败: {e}")
        
        with sync_playwright() as p:
            browser = p.chromium.launch_persistent_context(
                user_data_dir=user_data_dir,
                headless=self.headless,
                args=["--disable-blink-features=AutomationControlled"]
            )
            
            try:
                page = browser.pages[0] if browser.pages else browser.new_page()
                page.set_default_timeout(self.timeout)
                
                # 注册响应拦截器
                page.on("response", handle_response)
                
                self.logger.info("正在打开 DeepSeek 首页...")
                page.goto("https://chat.deepseek.com/")
                
                # 检查是否需要登录
                time.sleep(2)
                if "login" in page.url or page.query_selector("text=登录"):
                    self.logger.warning("检测到可能需要登录，请在浏览器窗口中完成登录...")
                    try:
                        page.wait_for_url("**/chat.deepseek.com/**", timeout=120000)
                    except:
                        self.logger.error("登录超时，请确保已手动登录并保存状态。")
                
                # 1. 等待输入框加载并输入
                page.wait_for_selector("textarea", timeout=self.timeout)
                page.click("textarea")
                time.sleep(0.5)
                page.fill("textarea", prompt)
                self.logger.info(f"已输入提问: {prompt[:50]}...")
                time.sleep(1)
                
                # 2. 开启"联网搜索" - 智能判断状态
                try:
                    # 寻找包含“联网搜索”文字的按钮容器
                    # DeepSeek 的开关通常是一个包含图标和文字的 div
                    search_toggle = page.locator("div:has-text('联网搜索')").last
                    
                    if search_toggle.is_visible():
                        # 检查是否已经激活
                        # 逻辑：检查该元素或其父级是否包含特定的激活类名（如 'checked' 或颜色相关的类）
                        # 或者检查其内部是否有特定的激活样式
                        is_active = False
                        
                        # 方案 A: 检查 class 中是否包含 'checked' 或 'active' (DeepSeek 常用)
                        class_attr = search_toggle.get_attribute("class") or ""
                        parent_class = ""
                        try:
                            parent_class = page.evaluate("el => el.parentElement.className", search_toggle.element_handle())
                        except: pass
                        
                        if "checked" in class_attr.lower() or "active" in class_attr.lower() or "checked" in str(parent_class).lower():
                            is_active = True
                        
                        # 方案 B: 检查颜色（激活时通常是蓝色 #247fff 或类似）
                        if not is_active:
                            color = page.evaluate("el => window.getComputedStyle(el).color", search_toggle.element_handle())
                            # 如果颜色不是默认的灰色（比如变成了蓝色），则认为已激活
                            if "rgb(36, 127, 255)" in color or "rgb(0, 0, 0)" not in color: # 简单判断非黑即活
                                is_active = True

                        if is_active:
                            self.logger.info("检测到‘联网搜索’已默认开启，跳过点击。")
                        else:
                            search_toggle.click()
                            self.logger.info("已手动开启‘联网搜索’")
                            time.sleep(0.5)
                except Exception as e:
                    self.logger.warning(f"判断联网搜索状态失败: {e}")
                
                # 3. 点击发送按钮
                try:
                    # 根据最新 UI，发送按钮是一个蓝色的圆形图标按钮
                    # 尝试多个可能的选择器
                    send_selectors = [
                        "div[class*='_7436f3']", # 常见的类名模式
                        "button:has(svg)",       # 包含图标的按钮
                        ".ds-icon--send",        # 图标类名
                        "div[role='button'] >> svg" # 角色为按钮的 div 下的 svg
                    ]
                    
                    sent = False
                    for selector in send_selectors:
                        btn = page.locator(selector).last
                        if btn.is_visible() and btn.is_enabled():
                            btn.click()
                            sent = True
                            self.logger.info(f"通过选择器 {selector} 点击了发送按钮")
                            break
                    
                    if not sent:
                        # 备选：使用键盘快捷键
                        page.keyboard.press("Enter") # 或者 Control+Enter
                        self.logger.info("已通过 Enter 键发送")
                        
                except Exception as e:
                    self.logger.warning(f"点击发送按钮失败: {e}")
                    page.keyboard.press("Control+Enter")
                
                self.logger.info("已发送提问，等待 AI 回答...")
                
                # 4. 等待回答生成完成
                time.sleep(5)  # 等待请求发送
                
                # 等待回答容器出现
                content_selector = ".ds-markdown"
                try:
                    page.wait_for_selector(content_selector, timeout=self.timeout)
                except:
                    self.logger.warning("未发现 .ds-markdown 容器")
                
                # 循环检查生成状态
                max_retries = 30
                last_content = ""
                for i in range(max_retries):
                    time.sleep(2)
                    try:
                        # 尝试获取当前内容
                        content_el = page.query_selector(content_selector)
                        if content_el:
                            current_content = content_el.inner_text()
                            
                            # 检查是否生成完成
                            if len(current_content) > 100:
                                if current_content == last_content:
                                    # 内容不再变化，检查是否有"停止生成"按钮
                                    stop_btn = page.query_selector("text=停止生成")
                                    if not stop_btn:
                                        self.logger.info("回答生成已完成")
                                        full_response_text = current_content
                                        break
                                
                            last_content = current_content
                            self.logger.info(f"正在生成中... (当前长度: {len(current_content)}, 已捕获 {len(captured_search_results)} 个搜索结果)")
                    except Exception as e:
                        continue
                
                # 5. 数据已从网络接口抓取完成，优先使用接口数据
                if len(captured_search_results) == 0:
                    self.logger.warning("未通过 API 接口抓取到引用，尝试从 DOM 提取作为补充...")
                    api_captured_urls = set()
                else:
                    self.logger.info(f"已通过 API 接口抓取到 {len(captured_search_results)} 个引用")
                    api_captured_urls = {r.get('url', '') for r in captured_search_results if r.get('url')}
                
                # 如果接口没有抓取到数据，尝试从 DOM 提取作为最后手段
                if len(captured_search_results) == 0:
                    try:
                        # 尝试多种方式提取引用链接
                        # DeepSeek 使用 ds-markdown-cite 类标记引用
                        # 优先提取带引用标记的链接
                        link_selectors = [
                            ".ds-markdown a[href^='http'] .ds-markdown-cite",  # 优先：带引用标记的链接
                            ".ds-markdown a[href^='https'] .ds-markdown-cite",
                            ".ds-markdown a[href^='http']",  # markdown 内容中的所有链接
                            ".ds-markdown a[href^='https']",
                            "a[href^='http'] .ds-markdown-cite",  # 所有带引用标记的链接
                            "a[href^='https'] .ds-markdown-cite",
                            "a[href^='http']",  # 所有外部链接
                            "a[href^='https']",
                            "[class*='citation'] a",  # 引用相关的链接
                            "[class*='reference'] a",
                            "[class*='source'] a",  # 来源相关的链接
                        ]
                        
                        seen_dom_urls = set(api_captured_urls)  # 从 API 已捕获的 URL 开始
                        dom_extracted_count = 0
                        
                        for selector in link_selectors:
                            try:
                                links = page.query_selector_all(selector)
                                self.logger.debug(f"选择器 '{selector}' 找到 {len(links)} 个链接")
                                
                                for link in links:
                                    try:
                                        # 如果选择器匹配的是 .ds-markdown-cite，需要找到父链接
                                        link_tag = link.evaluate("el => el.tagName.toLowerCase()")
                                        link_class = link.get_attribute("class") or ""
                                        
                                        if link_tag == 'span' or 'ds-markdown-cite' in link_class:
                                            # 找到父级 a 标签
                                            try:
                                                parent_a = link.evaluate_handle("el => el.closest('a')")
                                                if parent_a:
                                                    link = parent_a
                                                else:
                                                    # 如果找不到父 a，跳过这个元素
                                                    continue
                                            except:
                                                continue
                                        
                                        href = link.get_attribute("href")
                                        if not href:
                                            continue
                                        
                                        # 过滤掉 DeepSeek 自己的域名
                                        if any(d in href.lower() for d in ["deepseek.com", "deepseek.ai"]):
                                            continue
                                        
                                        # 去重
                                        if href in seen_dom_urls:
                                            continue
                                        seen_dom_urls.add(href)
                                        
                                        # 提取引用序号（关键修复：从 ds-markdown-cite 中提取）
                                        cite_index = 0
                                        try:
                                            # 查找链接内的 ds-markdown-cite 元素
                                            cite_element = link.query_selector(".ds-markdown-cite")
                                            if cite_element:
                                                # 从 cite 元素中提取序号
                                                cite_text = cite_element.inner_text().strip()
                                                # 尝试从文本中提取数字（如 "1", "2"）
                                                import re
                                                match = re.search(r'\d+', cite_text)
                                                if match:
                                                    cite_index = int(match.group())
                                                else:
                                                    # 尝试从 span 的绝对定位元素中提取
                                                    cite_number = cite_element.evaluate("""
                                                        el => {
                                                            let spans = el.querySelectorAll('span');
                                                            for (let span of spans) {
                                                                let text = span.textContent.trim();
                                                                let num = parseInt(text);
                                                                if (!isNaN(num) && num > 0) {
                                                                    return num;
                                                                }
                                                            }
                                                            return 0;
                                                        }
                                                    """)
                                                    cite_index = cite_number or 0
                                        except Exception as e:
                                            self.logger.debug(f"提取引用序号失败: {e}")
                                        
                                        # 如果没有找到序号，尝试从链接周围的文本中提取
                                        if cite_index == 0:
                                            try:
                                                # 查找链接前的引用标记
                                                prev_text = link.evaluate("""
                                                    el => {
                                                        let text = el.textContent || '';
                                                        let match = text.match(/\\[(\\d+)\\]/);
                                                        if (match) return parseInt(match[1]);
                                                        
                                                        // 查找父元素中的引用标记
                                                        let parent = el.parentElement;
                                                        if (parent) {
                                                            let parentText = parent.textContent || '';
                                                            let parentMatch = parentText.match(/\\[(\\d+)\\]/);
                                                            if (parentMatch) return parseInt(parentMatch[1]);
                                                        }
                                                        return 0;
                                                    }
                                                """)
                                                cite_index = prev_text or 0
                                            except:
                                                pass
                                        
                                        # 如果还是没有找到序号，使用当前计数
                                        if cite_index == 0:
                                            cite_index = len(captured_search_results) + 1
                                        
                                        # 提取标题
                                        title = link.inner_text().strip()
                                        # 移除引用标记（如 [1]）从标题中
                                        import re
                                        title = re.sub(r'\[\d+\]', '', title).strip()
                                        
                                        if not title:
                                            # 尝试从父元素或附近元素获取
                                            try:
                                                parent_text = link.evaluate("""
                                                    el => {
                                                        let parent = el.parentElement;
                                                        if (parent) {
                                                            let text = parent.textContent || '';
                                                            // 移除引用标记
                                                            text = text.replace(/\\[\\d+\\]/g, '').trim();
                                                            return text.substring(0, 100);
                                                        }
                                                        return '';
                                                    }
                                                """)
                                                title = parent_text
                                            except:
                                                pass
                                        
                                        # 提取摘要（尝试从附近元素）
                                        snippet = ""
                                        try:
                                            sibling_text = link.evaluate("""
                                                el => {
                                                    let next = el.nextElementSibling;
                                                    if (next && next.textContent) {
                                                        return next.textContent.trim().substring(0, 200);
                                                    }
                                                    let parent = el.parentElement;
                                                    if (parent && parent.nextElementSibling) {
                                                        return parent.nextElementSibling.textContent.trim().substring(0, 200);
                                                    }
                                                    return '';
                                                }
                                            """)
                                            snippet = sibling_text
                                        except:
                                            pass
                                        
                                        captured_search_results.append({
                                            "url": href,
                                            "title": title or extract_domain(href),
                                            "snippet": snippet,
                                            "site_name": extract_domain(href),
                                            "cite_index": cite_index
                                        })
                                        dom_extracted_count += 1
                                        self.logger.debug(f"从 DOM 捕获引用: {href[:50]}... (cite_index: {cite_index})")
                                    except Exception as e:
                                        self.logger.debug(f"提取链接失败: {e}")
                                        continue
                            except Exception as e:
                                self.logger.debug(f"选择器 '{selector}' 执行失败: {e}")
                                continue
                    
                        self.logger.info(f"从 DOM 提取到 {dom_extracted_count} 个新引用链接（API 已捕获 {len(api_captured_urls)} 个）")
                        
                        # 尝试查找引用列表区域（DeepSeek 可能在底部或侧边显示引用列表）
                        try:
                            # 查找可能的引用列表容器
                            citation_containers = [
                                "[class*='citation']",
                                "[class*='reference']",
                                "[class*='source']",
                                "[class*='link-list']",
                                "[class*='reference-list']"
                            ]
                            
                            for container_selector in citation_containers:
                                try:
                                    containers = page.query_selector_all(container_selector)
                                    if containers:
                                        self.logger.debug(f"找到 {len(containers)} 个可能的引用容器: {container_selector}")
                                        for container in containers:
                                            # 在容器内查找链接
                                            container_links = container.query_selector_all("a[href^='http']")
                                            for link in container_links:
                                                try:
                                                    href = link.get_attribute("href")
                                                    if href and href not in seen_dom_urls:
                                                        seen_dom_urls.add(href)
                                                        title = link.inner_text().strip() or extract_domain(href)
                                                        captured_search_results.append({
                                                            "url": href,
                                                            "title": title,
                                                            "snippet": "",
                                                            "site_name": extract_domain(href),
                                                            "cite_index": len(captured_search_results) + 1
                                                        })
                                                        dom_extracted_count += 1
                                                except:
                                                    continue
                                except:
                                    continue
                        except Exception as e:
                            self.logger.debug(f"查找引用列表容器失败: {e}")
                    except Exception as e:
                        self.logger.warning(f"从 DOM 提取引用失败: {e}")
                
                # 6. 整理搜索结果（去重）
                seen_urls = set()
                unique_citations = []
                for result in captured_search_results:
                    url = result.get('url', '')
                    if url and url not in seen_urls:
                        seen_urls.add(url)
                        unique_citations.append({
                            "url": url,
                            "title": result.get('title', ''),
                            "snippet": result.get('snippet', ''),
                            "site_name": result.get('site_name', ''),
                            "cite_index": result.get('cite_index', 0)
                        })
                
                # 按 cite_index 排序
                unique_citations.sort(key=lambda x: x.get('cite_index', 999))
                
                # 计算数据来源统计
                api_captured_count = len(api_captured_urls)
                dom_extracted_count = len(unique_citations) - api_captured_count
                if dom_extracted_count < 0:
                    dom_extracted_count = 0
                
                # 数据捕获汇总日志
                self.logger.info("")
                self.logger.info("=" * 60)
                self.logger.info("📊 数据捕获汇总")
                self.logger.info("=" * 60)
                
                # 查询信息汇总
                self.logger.info(f"🔍 查询信息 (共 {len(captured_queries)} 个):")
                if captured_queries:
                    for idx, q in enumerate(captured_queries, 1):
                        self.logger.info(f"  {idx}. \"{q}\"")
                else:
                    self.logger.info("  (未捕获到查询)")
                
                # 网站信息汇总
                self.logger.info("")
                self.logger.info(f"🌐 抓取网站 (共 {len(unique_citations)} 个唯一网站):")
                self.logger.info(f"  - API 拦截: {api_captured_count} 个")
                self.logger.info(f"  - DOM 提取: {dom_extracted_count} 个")
                
                if unique_citations:
                    # 按域名分组统计
                    domain_count = {}
                    for cite in unique_citations:
                        domain = cite.get('site_name', 'unknown')
                        domain_count[domain] = domain_count.get(domain, 0) + 1
                    
                    self.logger.info("")
                    self.logger.info("  网站列表 (前15个):")
                    for cite in unique_citations[:15]:
                        cite_index = cite.get('cite_index', 0)
                        site_name = cite.get('site_name', 'unknown')
                        title = cite.get('title', '')[:40] or '(无标题)'
                        url = cite.get('url', '')[:50]
                        self.logger.info(f"    [{cite_index}] {site_name}: {title}... ({url}...)")
                    
                    if len(unique_citations) > 15:
                        self.logger.info(f"    ... 还有 {len(unique_citations) - 15} 个网站未显示")
                    
                    self.logger.info("")
                    self.logger.info("  域名分布 (前10个):")
                    sorted_domains = sorted(domain_count.items(), key=lambda x: x[1], reverse=True)
                    for domain, count in sorted_domains[:10]:
                        self.logger.info(f"    {domain}: {count} 次")
                else:
                    self.logger.info("  (未捕获到网站)")
                
                self.logger.info("")
                self.logger.info("=" * 60)
                self.logger.info("✅ 数据捕获完成")
                self.logger.info(f"   - 查询: {len(captured_queries)} 个")
                self.logger.info(f"   - 网站: {len(unique_citations)} 个")
                self.logger.info("=" * 60)
                self.logger.info("")
                
                # 如果捕获数量明显少于预期，输出调试信息
                if len(unique_citations) < 3:
                    self.logger.warning("⚠️ 捕获到的引用数量较少，可能存在问题")
                    self.logger.info("💡 调试建议：")
                    self.logger.info("   1. 检查页面中是否确实显示了引用链接")
                    self.logger.info("   2. 查看浏览器开发者工具的 Network 标签，找到 API 响应")
                    self.logger.info("   3. 检查页面 HTML 中引用链接的实际结构")
                
                return {
                    "full_text": full_response_text or last_content,
                    "queries": captured_queries,  # 拓展词
                    "citations": unique_citations  # 参考网页
                }
            finally:
                browser.close()
