import os
import json
import time
import requests
from typing import Dict, Any, List
from providers.base import BaseProvider
from core.parser import extract_domain


class BochaApiProvider(BaseProvider):
    """
    博查 (Bocha AI) API Provider
    使用 Bocha AI Web Search API 进行搜索
    API 文档: https://bocha-ai.feishu.cn/wiki/HmtOw1z6vik14Fkdu5uc9VaInBb
    API 端点: https://api.bocha.cn/v1/web-search
    """
    
    def __init__(self, headless: bool = False, timeout: int = 30000):
        super().__init__(headless=headless, timeout=timeout)
        self.api_key = os.getenv("BOCHA_API_KEY")
        self.api_base_url = os.getenv("BOCHA_API_BASE_URL", "https://api.bocha.cn")
        
        if not self.api_key:
            self.logger.warning("BOCHA_API_KEY 环境变量未设置，搜索功能可能无法正常工作")
    
    def search(self, keyword: str, prompt: str) -> Dict[str, Any]:
        """
        使用博查 API 进行搜索
        
        Args:
            keyword: 搜索关键词
            prompt: 提示词（通常与 keyword 相同）
        
        Returns:
            包含 full_text, queries, citations 的字典
        """
        if not self.api_key:
            raise ValueError("BOCHA_API_KEY 环境变量未设置，无法使用博查 API")
        
        self.logger.info(f"🔍 开始使用博查 API 搜索: {keyword}")
        
        try:
            # 构建 API 请求
            # API 文档: https://bocha-ai.feishu.cn/wiki/HmtOw1z6vik14Fkdu5uc9VaInBb
            api_url = f"{self.api_base_url}/v1/web-search"
            
            headers = {
                "Authorization": f"Bearer {self.api_key}",
                "Content-Type": "application/json"
            }
            
            # 构建请求 payload
            payload = {
                "query": prompt or keyword,
                "summary": True,
                "freshness": "noLimit",
                "count": 10
            }
            
            self.logger.info(f"📡 发送请求到: {api_url}")
            self.logger.debug(f"请求参数: {json.dumps(payload, ensure_ascii=False)}")
            
            # 发送 HTTP 请求
            response = requests.post(
                api_url,
                headers=headers,
                json=payload,
                timeout=self.timeout / 1000  # 转换为秒
            )
            
            response.raise_for_status()
            data = response.json()
            
            self.logger.info(f"✅ 收到博查 API 响应")
            self.logger.debug(f"响应数据结构: {list(data.keys()) if isinstance(data, dict) else '非字典类型'}")
            
            # 解析响应数据
            # 根据实际的 API 响应格式调整解析逻辑
            full_text = ""
            queries = []
            citations = []
            
            # 提取回答文本（摘要）
            # 根据博查 API 文档，响应可能包含 summary 字段
            if "summary" in data:
                summary_data = data.get("summary")
                if isinstance(summary_data, str):
                    full_text = summary_data
                elif isinstance(summary_data, dict):
                    full_text = summary_data.get("text", summary_data.get("content", summary_data.get("summary", "")))
            elif "answer" in data:
                full_text = data.get("answer", "")
            elif "content" in data:
                full_text = data.get("content", "")
            elif "text" in data:
                full_text = data.get("text", "")
            elif "response" in data:
                response_data = data.get("response", {})
                if isinstance(response_data, str):
                    full_text = response_data
                elif isinstance(response_data, dict):
                    full_text = response_data.get("text", response_data.get("content", ""))
            
            # 提取查询词（拓展词）
            if "queries" in data:
                queries = data.get("queries", [])
                if isinstance(queries, str):
                    queries = [queries]
            elif "search_queries" in data:
                queries = data.get("search_queries", [])
                if isinstance(queries, str):
                    queries = [queries]
            
            # 提取引用（搜索结果）
            # 根据博查 API 文档，响应可能包含 results 或 items 字段
            if "results" in data:
                results = data.get("results", [])
                for idx, result in enumerate(results):
                    if isinstance(result, dict):
                        url = result.get("url", result.get("link", result.get("href", "")))
                        if url:
                            citations.append({
                                "url": url,
                                "title": result.get("title", result.get("name", "")),
                                "snippet": result.get("snippet", result.get("description", result.get("summary", ""))),
                                "site_name": result.get("site_name", result.get("source", extract_domain(url))),
                                "cite_index": result.get("cite_index", result.get("index", idx + 1)),
                                "query_indexes": result.get("query_indexes", [])
                            })
            elif "items" in data:
                items = data.get("items", [])
                for idx, item in enumerate(items):
                    if isinstance(item, dict):
                        url = item.get("url", item.get("link", item.get("href", "")))
                        if url:
                            citations.append({
                                "url": url,
                                "title": item.get("title", item.get("name", "")),
                                "snippet": item.get("snippet", item.get("description", item.get("summary", ""))),
                                "site_name": item.get("site_name", item.get("source", extract_domain(url))),
                                "cite_index": item.get("cite_index", item.get("index", idx + 1)),
                                "query_indexes": item.get("query_indexes", [])
                            })
            elif "citations" in data and isinstance(data.get("citations"), list):
                citations_data = data.get("citations", [])
                for idx, cite in enumerate(citations_data):
                    if isinstance(cite, dict):
                        url = cite.get("url", cite.get("link", cite.get("href", "")))
                        if url:
                            citations.append({
                                "url": url,
                                "title": cite.get("title", cite.get("name", "")),
                                "snippet": cite.get("snippet", cite.get("description", cite.get("summary", ""))),
                                "site_name": cite.get("site_name", cite.get("source", extract_domain(url))),
                                "cite_index": cite.get("cite_index", cite.get("index", idx + 1)),
                                "query_indexes": cite.get("query_indexes", [])
                            })
            elif "references" in data:
                references = data.get("references", [])
                for idx, ref in enumerate(references):
                    if isinstance(ref, dict):
                        url = ref.get("url", ref.get("link", ref.get("href", "")))
                        if url:
                            citations.append({
                                "url": url,
                                "title": ref.get("title", ref.get("name", "")),
                                "snippet": ref.get("snippet", ref.get("description", ref.get("summary", ""))),
                                "site_name": ref.get("site_name", ref.get("source", extract_domain(url))),
                                "cite_index": ref.get("cite_index", ref.get("index", idx + 1)),
                                "query_indexes": ref.get("query_indexes", [])
                            })
            
            # 如果没有提取到查询词，使用原始关键词
            if not queries:
                queries = [keyword] if keyword else []
            
            # 日志输出
            self.logger.info(f"\n{'='*60}")
            self.logger.info(f"📊 博查 API 数据捕获汇总")
            self.logger.info(f"{'='*60}")
            self.logger.info(f"🔍 查询信息 (共 {len(queries)} 个):")
            for idx, q in enumerate(queries, 1):
                self.logger.info(f"  {idx}. \"{q}\"")
            
            self.logger.info(f"\n🌐 抓取网站 (共 {len(citations)} 个):")
            for cite in citations[:10]:  # 只显示前10个
                self.logger.info(f"  [{cite.get('cite_index', 0)}] {cite.get('site_name', 'unknown')}: {cite.get('title', '')[:50]}...")
            
            if len(citations) > 10:
                self.logger.info(f"  ... 还有 {len(citations) - 10} 个网站未显示")
            
            self.logger.info(f"\n📝 回答文本长度: {len(full_text)} 字符")
            if full_text:
                self.logger.info(f"   文本预览: {full_text[:100]}...")
            
            self.logger.info(f"{'='*60}\n")
            
            return {
                "full_text": full_text,
                "queries": queries,
                "citations": citations
            }
            
        except requests.exceptions.RequestException as e:
            error_msg = f"博查 API 请求失败: {str(e)}"
            self.logger.error(error_msg, exc_info=True)
            raise Exception(error_msg)
        except json.JSONDecodeError as e:
            error_msg = f"博查 API 响应解析失败: {str(e)}"
            self.logger.error(error_msg, exc_info=True)
            raise Exception(error_msg)
        except Exception as e:
            error_msg = f"博查 API 搜索失败: {str(e)}"
            self.logger.error(error_msg, exc_info=True)
            raise Exception(error_msg)
