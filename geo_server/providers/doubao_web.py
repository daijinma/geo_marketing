import time
import os
import json
import re
from playwright.sync_api import sync_playwright
from providers.base import BaseProvider
from core.parser import extract_domain
from utils.encoding import ensure_utf8_string


class DoubaoWebProvider(BaseProvider):
    def search(self, keyword: str, prompt: str):
        user_data_dir = os.path.join(os.getenv("BROWSER_DATA_DIR", "./browser_data"), "doubao")
        
        # 用于存储拦截到的数据
        captured_queries = []
        captured_search_results = []
        full_response_text = ""
        
        def handle_response(response):
            """拦截豆包的 API 响应，提取搜索结果和拓展词"""
            nonlocal captured_search_results, captured_queries, full_response_text
            
            # 豆包的 API 端点：
            # - /chat/completion (主要端点)
            # - /api/chat/stream
            # - /api/v1/chat/completions
            # - /api/bot/chat
            # - /api/chat
            url_lower = response.url.lower()
            if any(endpoint in url_lower for endpoint in ["/chat/completion", "/api/chat", "/api/v1/chat", "/api/bot/chat", "/stream"]):
                try:
                    content_type = response.headers.get("content-type", "")
                    self.logger.info(f"🔍 拦截到豆包 API 响应: {response.url[:150]}")
                    self.logger.info(f"   Content-Type: {content_type}")
                    
                    # 处理 SSE 流
                    if "text/event-stream" in content_type or "stream" in url_lower or "/chat/completion" in url_lower:
                        try:
                            body = response.text()
                            self.logger.info(f"📡 处理豆包 SSE 流响应")
                            
                            # 解析 SSE 数据流
                            line_count = 0
                            data_count = 0
                            for line in body.split('\n'):
                                line_count += 1
                                if line.startswith('data: '):
                                    try:
                                        json_str = line[6:].strip()  # 去掉 "data: " 前缀
                                        if json_str and json_str != '[DONE]' and json_str != 'null':
                                            data = json.loads(json_str)
                                            data_count += 1
                                            
                                            # 记录数据结构信息（用于调试）
                                            if data_count <= 3:  # 只记录前3个数据包，避免日志过多
                                                self.logger.info(f"📦 数据包 #{data_count} 结构: {list(data.keys())}")
                                            
                                            # 豆包的数据结构：patch_op 数组
                                            if 'patch_op' in data and isinstance(data['patch_op'], list):
                                                self.logger.info(f"   ✅ 发现 patch_op 数组，包含 {len(data['patch_op'])} 个 patch")
                                                
                                                for patch_idx, patch in enumerate(data['patch_op']):
                                                    patch_object = patch.get('patch_object')
                                                    patch_type = patch.get('patch_type')
                                                    
                                                    self.logger.info(f"   🔹 Patch #{patch_idx}: object={patch_object}, type={patch_type}")
                                                    
                                                    # patch_object: 1 表示消息内容
                                                    if patch_object == 1 and patch_type == 1:
                                                        patch_value = patch.get('patch_value', {})
                                                        
                                                        # 查找 content_block
                                                        if 'content_block' in patch_value:
                                                            blocks = patch_value['content_block']
                                                            self.logger.info(f"      📚 发现 {len(blocks)} 个 content_block")
                                                            
                                                            for block_idx, block in enumerate(blocks):
                                                                block_type = block.get('block_type')
                                                                self.logger.info(f"      🔸 Block #{block_idx}: type={block_type}")
                                                                
                                                                # block_type: 10000 表示文本块
                                                                if block_type == 10000:
                                                                    self.logger.info(f"         ✅ 文本块 (block_type=10000)")
                                                                    content = block.get('content', {})
                                                                    text_block = content.get('text_block', {})
                                                                    
                                                                    # 提取文本内容
                                                                    if 'text' in text_block:
                                                                        text_content = text_block.get('text', '')
                                                                        if text_content:
                                                                            text_content_encoded = ensure_utf8_string(text_content, self.logger)
                                                                            full_response_text += text_content_encoded
                                                                            self.logger.debug(f"         📝 提取文本: {text_content_encoded[:50]}...")
                                                                
                                                                # block_type: 10025 表示搜索查询结果块
                                                                elif block_type == 10025:
                                                                    self.logger.info(f"         ✅ 搜索查询结果块 (block_type=10025)")
                                                                    content = block.get('content', {})
                                                                    search_block = content.get('search_query_result_block', {})
                                                                    
                                                                    # 提取查询词
                                                                    if 'queries' in search_block:
                                                                        queries = search_block.get('queries', [])
                                                                        self.logger.info(f"         🔍 发现 {len(queries)} 个查询词")
                                                                        for q in queries:
                                                                            if isinstance(q, str):
                                                                                q_encoded = ensure_utf8_string(q, self.logger)
                                                                                if q_encoded not in captured_queries:
                                                                                    captured_queries.append(q_encoded)
                                                                                    self.logger.info(f"         📝 捕获查询: {q_encoded}")
                                                                            elif isinstance(q, dict):
                                                                                query_text = q.get('query', q.get('text', ''))
                                                                                if query_text:
                                                                                    query_text_encoded = ensure_utf8_string(query_text, self.logger)
                                                                                    if query_text_encoded not in captured_queries:
                                                                                        captured_queries.append(query_text_encoded)
                                                                                        self.logger.info(f"         📝 捕获查询: {query_text_encoded}")
                                                                    
                                                                    # 提取搜索结果
                                                                    if 'results' in search_block:
                                                                        results = search_block.get('results', [])
                                                                        self.logger.info(f"         📄 发现 {len(results)} 个搜索结果")
                                                                        
                                                                        for r_idx, r in enumerate(results):
                                                                            if isinstance(r, dict):
                                                                                # 检查是否有 text_card（网页链接）
                                                                                text_card = r.get('text_card', {})
                                                                                # 检查是否有 video_card（视频链接，如抖音）
                                                                                video_card = r.get('video_card', {})
                                                                                # 检查其他可能的卡片类型
                                                                                other_cards = {k: v for k, v in r.items() if k.endswith('_card') and k not in ['text_card', 'video_card']}
                                                                                
                                                                                if text_card:
                                                                                    url = text_card.get('url', '')
                                                                                    if url:
                                                                                        captured_search_results.append({
                                                                                            "url": ensure_utf8_string(url, self.logger),
                                                                                            "title": ensure_utf8_string(text_card.get('title', ''), self.logger),
                                                                                            "snippet": ensure_utf8_string(text_card.get('summary', ''), self.logger),
                                                                                            "site_name": ensure_utf8_string(text_card.get('sitename', ''), self.logger),
                                                                                            "cite_index": text_card.get('index', r.get('index', 0)),
                                                                                            "query_indexes": r.get('query_indexes', text_card.get('query_indexes', []))
                                                                                        })
                                                                                        self.logger.info(f"         🔗 捕获网页引用 #{r_idx+1}: {url[:80]}... (cite_index: {text_card.get('index', 0)})")
                                                                                        self.logger.info(f"            标题: {text_card.get('title', '')[:50]}...")
                                                                                        self.logger.info(f"            站点: {text_card.get('sitename', '')}")
                                                                                
                                                                                elif video_card:
                                                                                    # 处理视频卡片（如抖音视频）
                                                                                    video_url = video_card.get('url', '') or video_card.get('video_url', '')
                                                                                    if video_url:
                                                                                        captured_search_results.append({
                                                                                            "url": ensure_utf8_string(video_url, self.logger),
                                                                                            "title": ensure_utf8_string(video_card.get('title', video_card.get('description', '')), self.logger),
                                                                                            "snippet": ensure_utf8_string(video_card.get('description', video_card.get('summary', '')), self.logger),
                                                                                            "site_name": ensure_utf8_string(video_card.get('platform', 'video'), self.logger),
                                                                                            "cite_index": video_card.get('index', r.get('index', 0)),
                                                                                            "query_indexes": r.get('query_indexes', video_card.get('query_indexes', []))
                                                                                        })
                                                                                        self.logger.info(f"         🎬 捕获视频引用 #{r_idx+1}: {video_url[:80]}... (cite_index: {video_card.get('index', 0)})")
                                                                                        self.logger.info(f"            平台: {video_card.get('platform', 'unknown')}")
                                                                                        self.logger.info(f"            标题: {video_card.get('title', '')[:50]}...")
                                                                                
                                                                                elif other_cards:
                                                                                    # 记录其他类型的卡片（用于后续分析）
                                                                                    self.logger.info(f"         ⚠️  发现未处理的卡片类型: {list(other_cards.keys())}")
                                                                                    for card_type, card_data in other_cards.items():
                                                                                        self.logger.info(f"            {card_type}: {str(card_data)[:200]}...")
                                                                                
                                                                                else:
                                                                                    # 记录未识别的结果结构
                                                                                    self.logger.info(f"         ⚠️  结果 #{r_idx+1} 结构未识别: {list(r.keys())}")
                                                                                    self.logger.debug(f"            完整数据: {str(r)[:300]}...")
                                                                    
                                                                    # 记录 summary 信息
                                                                    if 'summary' in search_block:
                                                                        self.logger.info(f"         📊 搜索摘要: {search_block.get('summary', '')}")
                                                                
                                                                else:
                                                                    # 记录其他类型的 block
                                                                    self.logger.info(f"         ⚠️  未处理的 block_type: {block_type}")
                                                                    if block_idx < 2:  # 只记录前2个未处理的 block
                                                                        self.logger.debug(f"            Block 数据: {str(block)[:300]}...")
                                                    
                                                    else:
                                                        # 记录其他类型的 patch
                                                        if patch_idx < 3:  # 只记录前3个未处理的 patch
                                                            self.logger.info(f"      ⚠️  未处理的 patch: object={patch_object}, type={patch_type}")
                                            
                                            else:
                                                # 记录未识别的数据结构
                                                if data_count <= 3:
                                                    self.logger.info(f"   ⚠️  未识别 patch_op 结构，数据键: {list(data.keys())}")
                                                    # 检查是否有其他可能包含搜索结果的字段
                                                    for key in ['search', 'results', 'citations', 'references', 'videos', 'video']:
                                                        if key in data:
                                                            self.logger.info(f"      🔍 发现可能的搜索字段: {key}")
                                            
                                            # 兼容其他可能的数据结构（向后兼容）
                                            # 提取搜索查询词（多种可能的字段名）
                                            for query_field in ['search_queries', 'queries', 'search_query', 'query']:
                                                if query_field in data:
                                                    queries = data.get(query_field, [])
                                                    if isinstance(queries, list):
                                                        for q in queries:
                                                            if isinstance(q, dict):
                                                                query_text = q.get('query', q.get('text', ''))
                                                            else:
                                                                query_text = str(q)
                                                            if query_text:
                                                                query_text_encoded = ensure_utf8_string(query_text, self.logger)
                                                                if query_text_encoded not in captured_queries:
                                                                    captured_queries.append(query_text_encoded)
                                                    elif isinstance(queries, str):
                                                        queries_encoded = ensure_utf8_string(queries, self.logger)
                                                        if queries_encoded not in captured_queries:
                                                            captured_queries.append(queries_encoded)
                                            
                                            # 提取搜索结果（多种可能的字段名）
                                            for result_field in ['search_results', 'results', 'citations', 'references']:
                                                if result_field in data:
                                                    results = data.get(result_field, [])
                                                    if isinstance(results, list):
                                                        for r in results:
                                                            if isinstance(r, dict) and 'url' in r:
                                                                captured_search_results.append({
                                                                    "url": ensure_utf8_string(r.get('url', ''), self.logger),
                                                                    "title": ensure_utf8_string(r.get('title', r.get('name', '')), self.logger),
                                                                    "snippet": ensure_utf8_string(r.get('snippet', r.get('content', r.get('description', ''))), self.logger),
                                                                    "site_name": ensure_utf8_string(r.get('site_name', r.get('source', r.get('domain', ''))), self.logger),
                                                                    "cite_index": r.get('cite_index', r.get('index', r.get('order', 0))),
                                                                    "query_indexes": r.get('query_indexes', [])
                                                                })
                                            
                                            # 提取回答内容（多种可能的字段名）
                                            for content_field in ['content', 'text', 'message', 'answer']:
                                                if content_field in data:
                                                    content = data.get(content_field, '')
                                                    if content:
                                                        content_encoded = ensure_utf8_string(content, self.logger)
                                                        full_response_text += content_encoded
                                                elif 'delta' in data and content_field in data.get('delta', {}):
                                                    content = data['delta'].get(content_field, '')
                                                    if content:
                                                        content_encoded = ensure_utf8_string(content, self.logger)
                                                        full_response_text += content_encoded
                                            
                                            # 处理嵌套结构（如 data.message.content）
                                            if 'message' in data and isinstance(data['message'], dict):
                                                msg = data['message']
                                                if 'content' in msg:
                                                    content = msg['content']
                                                    if content:
                                                        content_encoded = ensure_utf8_string(content, self.logger)
                                                        full_response_text += content_encoded
                                            
                                    except json.JSONDecodeError as e:
                                        self.logger.debug(f"JSON 解析失败: {e}")
                                        continue
                        except Exception as e:
                            self.logger.debug(f"解析 SSE 响应失败: {e}")
                    
                    # 处理普通 JSON 响应
                    elif "application/json" in content_type:
                        try:
                            data = response.json()
                            
                            # 提取搜索相关信息
                            if 'search' in data:
                                search_data = data['search']
                                if 'queries' in search_data:
                                    queries = search_data['queries']
                                    if isinstance(queries, list):
                                        encoded_queries = [ensure_utf8_string(q if isinstance(q, str) else q.get('query', ''), self.logger) for q in queries]
                                        for q_encoded in encoded_queries:
                                            if q_encoded and q_encoded not in captured_queries:
                                                captured_queries.append(q_encoded)
                                if 'results' in search_data:
                                    for r in search_data['results']:
                                        if isinstance(r, dict) and 'url' in r:
                                            captured_search_results.append({
                                                "url": ensure_utf8_string(r.get('url', ''), self.logger),
                                                "title": ensure_utf8_string(r.get('title', ''), self.logger),
                                                "snippet": ensure_utf8_string(r.get('snippet', ''), self.logger),
                                                "site_name": ensure_utf8_string(r.get('source', ''), self.logger),
                                                "cite_index": r.get('index', 0),
                                                "query_indexes": r.get('query_indexes', [])
                                            })
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
                
                self.logger.info("正在打开豆包首页...")
                page.goto("https://www.doubao.com/")
                
                # 检查是否需要登录
                time.sleep(2)
                if "login" in page.url.lower() or page.query_selector("text=登录") or page.query_selector("text=立即登录"):
                    self.logger.warning("检测到可能需要登录，请在浏览器窗口中完成登录...")
                    try:
                        # 等待登录完成，URL 变化或登录按钮消失
                        page.wait_for_function("""
                            () => {
                                return !document.querySelector('text=登录') && 
                                       !document.querySelector('text=立即登录') &&
                                       window.location.href.includes('doubao.com');
                            }
                        """, timeout=120000)
                        self.logger.info("登录检测完成")
                    except:
                        self.logger.warning("登录检测超时，继续执行...")
                
                # 1. 等待输入框加载并输入
                # 尝试多种可能的选择器
                textarea_selectors = [
                    "textarea",
                    "textarea[placeholder*='输入']",
                    "textarea[placeholder*='提问']",
                    "[contenteditable='true']",
                    ".input-area textarea"
                ]
                
                textarea_found = False
                for selector in textarea_selectors:
                    try:
                        page.wait_for_selector(selector, timeout=5000)
                        page.click(selector)
                        time.sleep(0.5)
                        page.fill(selector, prompt)
                        textarea_found = True
                        self.logger.info(f"已输入提问: {prompt[:50]}...")
                        break
                    except:
                        continue
                
                if not textarea_found:
                    raise Exception("未找到输入框")
                
                time.sleep(1)
                
                # 2. 开启"联网搜索"或"深度搜索" - 智能判断状态
                try:
                    # 尝试多种可能的搜索开关选择器
                    search_toggle_selectors = [
                        "div:has-text('联网搜索')",
                        "div:has-text('深度搜索')",
                        "div:has-text('搜索')",
                        "button:has-text('搜索')",
                        "[aria-label*='搜索']",
                        "[title*='搜索']"
                    ]
                    
                    search_toggle = None
                    for selector in search_toggle_selectors:
                        try:
                            toggle = page.locator(selector).last
                            if toggle.is_visible():
                                search_toggle = toggle
                                break
                        except:
                            continue
                    
                    if search_toggle:
                        # 检查是否已经激活
                        is_active = False
                        
                        # 方案 A: 检查 class 中是否包含激活状态
                        try:
                            class_attr = search_toggle.get_attribute("class") or ""
                            parent_class = ""
                            try:
                                parent_class = page.evaluate("el => el.parentElement?.className || ''", search_toggle.element_handle())
                            except:
                                pass
                            
                            if any(keyword in (class_attr + parent_class).lower() for keyword in ["checked", "active", "on", "enabled"]):
                                is_active = True
                            
                            # 方案 B: 检查颜色或样式
                            if not is_active:
                                try:
                                    color = page.evaluate("el => window.getComputedStyle(el).color", search_toggle.element_handle())
                                    bg_color = page.evaluate("el => window.getComputedStyle(el).backgroundColor", search_toggle.element_handle())
                                    # 如果颜色不是默认的灰色，可能已激活
                                    if "rgb(0, 0, 0)" not in color and "rgb(128" not in color:
                                        is_active = True
                                except:
                                    pass
                            
                            if is_active:
                                self.logger.info("检测到'联网搜索'已默认开启，跳过点击。")
                            else:
                                search_toggle.click()
                                self.logger.info("已手动开启'联网搜索'")
                                time.sleep(0.5)
                        except Exception as e:
                            self.logger.debug(f"判断搜索开关状态失败: {e}")
                except Exception as e:
                    self.logger.debug(f"未找到搜索开关: {e}")
                
                # 3. 点击发送按钮
                try:
                    # 尝试多种可能的发送按钮选择器
                    send_selectors = [
                        # 精确匹配（优先）
                        "button#flow-end-msg-send",
                        "button[data-testid='chat_input_send_button']",
                        ".send-btn-wrapper button",
                        # 现有选择器（兜底）
                        "button[type='submit']",
                        "button:has(svg)",
                        "button:has-text('发送')",
                        "[aria-label*='发送']",
                        "[title*='发送']",
                        ".send-button",
                        "div[role='button']:has(svg)"
                    ]
                    
                    sent = False
                    for selector in send_selectors:
                        try:
                            btn = page.locator(selector).last
                            if btn.is_visible() and btn.is_enabled():
                                btn.click()
                                sent = True
                                self.logger.info(f"通过选择器 {selector} 点击了发送按钮")
                                break
                        except:
                            continue
                    
                    if not sent:
                        # 备选：使用键盘快捷键
                        page.keyboard.press("Enter")
                        self.logger.info("已通过 Enter 键发送")
                        
                except Exception as e:
                    self.logger.warning(f"点击发送按钮失败: {e}")
                    page.keyboard.press("Enter")
                
                self.logger.info("已发送提问，等待豆包回答...")
                
                # 4. 等待回答生成完成
                time.sleep(5)  # 等待请求发送
                
                # 等待回答容器出现（尝试多种选择器）
                content_selectors = [
                    "article",
                    ".message-content",
                    "[class*='message']",
                    "[class*='content']",
                    "[class*='answer']",
                    "[class*='response']",
                    ".chat-message"
                ]
                
                content_selector = None
                for selector in content_selectors:
                    try:
                        page.wait_for_selector(selector, timeout=10000)
                        content_selector = selector
                        self.logger.info(f"找到回答容器: {selector}")
                        break
                    except:
                        continue
                
                if not content_selector:
                    self.logger.warning("未发现标准回答容器，将尝试通用选择器")
                    content_selector = "body"
                
                # 循环检查生成状态
                max_retries = 30
                last_content = ""
                stable_count = 0
                for i in range(max_retries):
                    time.sleep(2)
                    try:
                        # 尝试获取当前内容
                        if content_selector == "body":
                            content_el = page.query_selector("body")
                        else:
                            content_el = page.query_selector(content_selector)
                        
                        if content_el:
                            current_content = content_el.inner_text()
                            
                            # 检查是否生成完成
                            if len(current_content) > 100:
                                if current_content == last_content:
                                    stable_count += 1
                                    if stable_count >= 2:  # 连续2次内容不变
                                        # 检查是否有"停止生成"按钮
                                        stop_btn = page.query_selector("text=停止生成") or page.query_selector("text=停止")
                                        if not stop_btn:
                                            self.logger.info("回答生成已完成")
                                            if not full_response_text:
                                                full_response_text = current_content
                                            break
                                else:
                                    stable_count = 0
                                
                                last_content = current_content
                                self.logger.info(f"正在生成中... (当前长度: {len(current_content)}, 已捕获 {len(captured_search_results)} 个搜索结果)")
                    except Exception as e:
                        self.logger.debug(f"检查生成状态失败: {e}")
                        continue
                
                # 5. 如果没有通过 API 拦截到引用，则从 DOM 提取
                if not captured_search_results:
                    self.logger.info("未通过 API 拦截到引用，尝试从页面提取...")
                    
                    # 尝试多种方式提取链接
                    link_selectors = [
                        "a[href^='http']",
                        "a[href^='https']",
                        "[class*='citation'] a",
                        "[class*='reference'] a",
                        "[class*='link'] a"
                    ]
                    
                    seen_dom_urls = set()
                    for selector in link_selectors:
                        try:
                            links = page.query_selector_all(selector)
                            for link in links:
                                try:
                                    href = link.get_attribute("href")
                                    if not href:
                                        continue
                                    
                                    # 过滤掉豆包自己的域名
                                    if any(d in href.lower() for d in ["doubao.com", "bytecheck.com", "volcengine.com", "bytedance.com"]):
                                        continue
                                    
                                    # 去重
                                    if href in seen_dom_urls:
                                        continue
                                    seen_dom_urls.add(href)
                                    
                                    # 提取标题
                                    title = link.inner_text().strip()
                                    if not title:
                                        # 尝试从父元素获取
                                        try:
                                            parent = link.evaluate("el => el.parentElement?.textContent || ''")
                                            title = parent.strip()[:100]
                                        except:
                                            pass
                                    
                                    # 提取摘要（尝试从附近元素）
                                    snippet = ""
                                    try:
                                        sibling = link.evaluate("""
                                            el => {
                                                let next = el.nextElementSibling;
                                                if (next && next.textContent) {
                                                    return next.textContent.trim().substring(0, 200);
                                                }
                                                return '';
                                            }
                                        """)
                                        snippet = sibling
                                    except:
                                        pass
                                    
                                    captured_search_results.append({
                                        "url": ensure_utf8_string(href, self.logger),
                                        "title": ensure_utf8_string(title or extract_domain(href), self.logger),
                                        "snippet": ensure_utf8_string(snippet, self.logger),
                                        "site_name": ensure_utf8_string(extract_domain(href), self.logger)
                                    })
                                except Exception as e:
                                    self.logger.debug(f"提取链接失败: {e}")
                                    continue
                        except:
                            continue
                
                # 6. 整理搜索结果（去重）并确保编码正确
                seen_urls = set()
                unique_citations = []
                for result in captured_search_results:
                    url = result.get('url', '')
                    if url and url not in seen_urls:
                        seen_urls.add(url)
                        # 确保所有文本字段都是正确的 UTF-8 编码
                        unique_citations.append({
                            "url": ensure_utf8_string(url, self.logger),
                            "title": ensure_utf8_string(result.get('title', ''), self.logger),
                            "snippet": ensure_utf8_string(result.get('snippet', ''), self.logger),
                            "site_name": ensure_utf8_string(result.get('site_name', ''), self.logger),
                            "cite_index": result.get('cite_index', 0)
                        })
                
                # 按 cite_index 排序
                unique_citations.sort(key=lambda x: x.get('cite_index', 999))
                
                # 打印拓展词
                self.logger.info(f"\n{'='*60}")
                self.logger.info(f"📊 豆包数据捕获汇总")
                self.logger.info(f"{'='*60}")
                self.logger.info(f"🔍 拓展搜索词: {len(captured_queries)} 个")
                for q in captured_queries:
                    self.logger.info(f"   - {q}")
                
                # 打印参考网页
                self.logger.info(f"\n📄 参考来源: {len(unique_citations)} 个")
                for cite in unique_citations:
                    cite_type = "🎬 视频" if any(video_domain in cite.get('url', '').lower() for video_domain in ['douyin.com', 'tiktok.com', 'video', 'bilibili.com']) else "🔗 网页"
                    self.logger.info(f"   [{cite.get('cite_index')}] {cite_type} {cite.get('site_name')}: {cite.get('title', '')[:50]}...")
                    self.logger.info(f"       URL: {cite.get('url', '')[:100]}...")
                
                # 如果捕获数量较少，提示检查日志
                if len(unique_citations) == 0:
                    self.logger.warning(f"\n⚠️  未捕获到任何引用，请检查上方的详细日志")
                    self.logger.info(f"💡 提示：查看日志中的 '⚠️' 标记，这些是未识别的数据结构")
                    self.logger.info(f"   请将这些数据结构信息提供给我，以便进一步优化解析逻辑")
                
                # 准备返回数据，确保文本编码正确
                final_full_text = full_response_text or last_content
                final_full_text = ensure_utf8_string(final_full_text, self.logger)
                
                # 确保 queries 中的文本也是正确的编码
                encoded_queries = [ensure_utf8_string(q, self.logger) for q in captured_queries]
                
                result_data = {
                    "full_text": final_full_text,
                    "queries": encoded_queries,  # 拓展词
                    "citations": unique_citations  # 参考网页
                }
                
                # 打印完整数据（JSON格式）
                self.logger.info(f"\n{'='*60}")
                self.logger.info(f"📋 完整数据输出 (JSON格式)")
                self.logger.info(f"{'='*60}")
                try:
                    # 创建可打印的数据副本（截断过长的文本）
                    # 确保文本是字符串类型，并正确处理编码
                    full_text = result_data["full_text"]
                    if isinstance(full_text, bytes):
                        # 如果是字节，尝试UTF-8解码
                        try:
                            full_text = full_text.decode('utf-8')
                        except UnicodeDecodeError:
                            # 如果UTF-8解码失败，尝试其他编码
                            try:
                                full_text = full_text.decode('utf-8', errors='replace')
                            except:
                                full_text = str(full_text)
                    elif not isinstance(full_text, str):
                        full_text = str(full_text)
                    
                    print_data = {
                        "full_text_length": len(full_text),
                        "full_text_preview": (full_text[:200] + "...") if len(full_text) > 200 else full_text,
                        "queries": result_data["queries"],
                        "citations": result_data["citations"]
                    }
                    # 使用 ensure_ascii=False 确保中文正确显示，并确保所有字符串都是UTF-8编码
                    json_str = json.dumps(print_data, ensure_ascii=False, indent=2)
                    self.logger.info(json_str)
                except Exception as e:
                    self.logger.warning(f"打印JSON数据失败: {e}")
                    import traceback
                    self.logger.debug(traceback.format_exc())
                
                # 打印完整文本长度信息
                full_text = result_data['full_text']
                if isinstance(full_text, bytes):
                    try:
                        full_text = full_text.decode('utf-8')
                    except UnicodeDecodeError:
                        full_text = full_text.decode('utf-8', errors='replace')
                elif not isinstance(full_text, str):
                    full_text = str(full_text)
                
                self.logger.info(f"\n📝 完整回答文本长度: {len(full_text)} 字符")
                if len(full_text) > 0:
                    self.logger.info(f"   文本预览: {full_text[:100]}...")
                
                self.logger.info(f"{'='*60}\n")
                
                return result_data
            finally:
                browser.close()
