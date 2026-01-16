"""
core/task_executor.py - 任务执行器
封装任务执行逻辑，支持多关键词、多平台的异步执行
"""
import os
import time
import logging
import threading
from typing import List, Dict, Any, Optional
from core.db import get_db_connection, update_domain_stats
from core.parser import extract_domain
from providers.deepseek_web import DeepSeekWebProvider
from providers.doubao_web import DoubaoWebProvider
from providers.bocha_api import BochaApiProvider

logger = logging.getLogger(__name__)


def save_to_db(keyword, platform, prompt, result, prompt_type="default", response_time_ms=None, error_message=None, task_id=None, task_query_id=None):
    """
    保存搜索结果到数据库（从 main.py 复用）
    
    Args:
        keyword: 搜索关键词
        platform: 平台名称
        prompt: 提示词
        result: 搜索结果
        prompt_type: 提示类型
        response_time_ms: 响应时间（毫秒）
        error_message: 错误信息
        task_id: task_jobs 表的 ID（可选，用于关联任务）
        task_query_id: task_query 表的 ID（可选，用于关联 executor_sub_query_log）
    """
    try:
        from providers.doubao_web import ensure_utf8_string
        
        with get_db_connection() as conn:
            cur = conn.cursor()
            
            # 确定搜索状态
            search_status = 'completed' if result and result.get("full_text") else 'failed'
            if error_message:
                search_status = 'failed'
            
            # 1. 插入搜索记录（包含任务关联字段）
            cur.execute("""
                INSERT INTO search_records 
                (keyword, platform, prompt_type, prompt, full_answer, response_time_ms, search_status, error_message, task_id, task_query_id) 
                VALUES (%s, %s, %s, %s, %s, %s, %s, %s, %s, %s) 
                RETURNING id
            """, (
                keyword, 
                platform, 
                prompt_type, 
                prompt,
                result.get("full_text", "") if result else "",
                response_time_ms,
                search_status,
                error_message,
                task_id,
                task_query_id
            ))
            record_id = cur.fetchone()[0]
            
            if not result:
                logger.warning(f"搜索失败，仅保存了记录 ID: {record_id}")
                return record_id, 0
            
            # 2. 插入拓展词 (带顺序)
            for idx, query in enumerate(result.get("queries", []), 1):
                cur.execute(
                    "INSERT INTO search_queries (record_id, query, query_order) VALUES (%s, %s, %s)",
                    (record_id, query, idx)
                )
            
            # 3. 插入引用 (利用唯一约束自动去重)，并保存 citation_id 用于关联
            citations_count = 0
            citation_ids = {}  # url -> citation_id 映射，用于后续关联 executor_sub_query_log
            for cite in result.get("citations", []):
                url = cite.get("url", "")
                if not url:
                    continue
                    
                domain = extract_domain(url)
                try:
                    # 先尝试插入，如果冲突则查询现有记录
                    cur.execute("""
                        INSERT INTO citations 
                        (record_id, cite_index, url, domain, title, snippet, site_name) 
                        VALUES (%s, %s, %s, %s, %s, %s, %s)
                        ON CONFLICT (record_id, url) DO NOTHING
                        RETURNING id
                    """, (
                        record_id, 
                        cite.get("cite_index", 0), 
                        url, 
                        domain, 
                        cite.get("title", ""), 
                        cite.get("snippet", ""), 
                        cite.get("site_name", "")
                    ))
                    
                    row = cur.fetchone()
                    if row:
                        # 新插入的记录
                        citation_id = row[0]
                        citations_count += 1
                        citation_ids[url] = citation_id
                        # 更新域名统计
                        update_domain_stats(conn, domain, platform)
                    else:
                        # 记录已存在，查询 citation_id
                        cur.execute("""
                            SELECT id FROM citations 
                            WHERE record_id = %s AND url = %s
                            LIMIT 1
                        """, (record_id, url))
                        row = cur.fetchone()
                        if row:
                            citation_ids[url] = row[0]
                        
                except Exception as e:
                    logger.debug(f"插入引用失败: {e}")
            
            # 4. 如果提供了 task_query_id，保存到 executor_sub_query_log 表
            # 合并策略：取消 A 类型（只有 sub_query），合并到 B 类型（有 url）
            # 根据 citation 的 query_indexes 字段建立真实关联
            if task_query_id and result:
                queries = result.get("queries", [])
                citations = result.get("citations", [])
                
                # 保存网址信息，根据 query_indexes 关联对应的 query
                # 如果 citations 为空但 queries 不为空，则不保存（取消 A 类型）
                for cite in citations:
                    url = cite.get("url", "")
                    if not url:
                        continue
                    
                    # 获取对应的 citation_id
                    citation_id = citation_ids.get(url)
                    
                    url = ensure_utf8_string(url) if isinstance(url, str) else url
                    domain = extract_domain(url)
                    title = ensure_utf8_string(cite.get("title", "")) if isinstance(cite.get("title", ""), str) else cite.get("title", "")
                    snippet = ensure_utf8_string(cite.get("snippet", "")) if isinstance(cite.get("snippet", ""), str) else cite.get("snippet", "")
                    site_name = ensure_utf8_string(cite.get("site_name", "")) if isinstance(cite.get("site_name", ""), str) else cite.get("site_name", "")
                    cite_index = cite.get("cite_index", 0)
                    
                    # 根据 query_indexes 获取对应的 query
                    # query_indexes[0] 表示关联到 queries 数组的第几个 query（索引从 0 开始）
                    sub_query = None
                    query_indexes = cite.get("query_indexes", [])
                    
                    if queries and query_indexes and len(query_indexes) > 0:
                        # 有明确的 query_indexes，按索引关联（DeepSeek 等情况）
                        query_idx = query_indexes[0]  # 只使用第一个索引
                        # 检查索引有效性
                        if isinstance(query_idx, int) and 0 <= query_idx < len(queries):
                            query = queries[query_idx]
                            if query:
                                sub_query = ensure_utf8_string(query) if isinstance(query, str) else query
                    elif queries and len(queries) == 1:
                        # 没有 query_indexes，但只有一个 query（豆包等情况）
                        # 认为所有链接都参考此 query
                        query = queries[0]
                        if query:
                            sub_query = ensure_utf8_string(query) if isinstance(query, str) else query
                    # 其他情况（没有 query_indexes 且 queries 不为 1 个）：保持现状，sub_query 为 NULL
                    
                    # 保存记录（每个 citation 只保存一次）
                    try:
                        cur.execute("""
                            INSERT INTO executor_sub_query_log 
                            (task_query_id, sub_query, record_id, url, domain, title, snippet, site_name, cite_index, citation_id)
                            VALUES (%s, %s, %s, %s, %s, %s, %s, %s, %s, %s)
                        """, (task_query_id, sub_query, record_id, url, domain, title, snippet, site_name, cite_index, citation_id))
                    except Exception as e:
                        logger.debug(f"插入网址信息失败: {e}")
            
            logger.info(f"✅ 成功保存 {platform} 的数据，记录 ID: {record_id}")
            logger.info(f"  - 拓展词: {len(result.get('queries', []))} 个")
            logger.info(f"  - 参考网页: {citations_count} 个")
            if response_time_ms:
                logger.info(f"  - 响应时间: {response_time_ms/1000:.2f} 秒")
            
            return record_id, citations_count
                
    except Exception as e:
        logger.error(f"❌ 保存到数据库失败: {e}", exc_info=True)
        return None, 0


def execute_single_task(keyword: str, platform: str, prompt: str, settings: Dict[str, Any], task_id: Optional[int] = None, task_query_id: Optional[int] = None) -> Dict[str, Any]:
    """
    执行单个关键词-平台组合的搜索任务
    
    Args:
        keyword: 搜索关键词
        platform: 平台名称 (deepseek, doubao, bocha)
        prompt: 提示词
        settings: 设置字典 (headless, timeout等)
        task_id: task_jobs 表的 ID（可选，用于关联任务）
        task_query_id: task_query 表的 ID（可选，用于关联 executor_sub_query_log）
    
    Returns:
        包含执行结果的字典
    """
    headless = settings.get("headless", False)
    timeout = settings.get("timeout", 60000)
    
    providers = {
        "deepseek": DeepSeekWebProvider(headless=headless, timeout=timeout),
        "doubao": DoubaoWebProvider(headless=headless, timeout=timeout),
        "bocha": BochaApiProvider(headless=headless, timeout=timeout)
    }
    
    # 平台名称规范化
    platform_lower = platform.lower().strip()
    matched_platform = None
    for key in providers.keys():
        if key.lower() == platform_lower:
            matched_platform = key
            break
    
    if not matched_platform:
        return {
            "keyword": keyword,
            "platform": platform,
            "status": "failed",
            "error_message": f"未找到平台 [{platform}] 的 Provider",
            "record_id": None,
            "citations_count": 0
        }
    
    provider = providers[matched_platform]
    logger.info(f"\n{'='*60}")
    logger.info(f"🚀 开始执行任务: [{keyword}] 在平台 [{matched_platform}]")
    logger.info(f"{'='*60}")
    
    start_time = time.time()
    result = None
    error_message = None
    
    try:
        result = provider.search(keyword, prompt)
        response_time_ms = int((time.time() - start_time) * 1000)
        
        if result and result.get("full_text"):
            record_id, citations_count = save_to_db(
                keyword, matched_platform, prompt, result, 
                prompt_type="api_task", 
                response_time_ms=response_time_ms,
                task_id=task_id,
                task_query_id=task_query_id
            )
            logger.info(f"✅ {matched_platform} 任务完成")
            return {
                "keyword": keyword,
                "platform": matched_platform,
                "status": "completed",
                "record_id": record_id,
                "citations_count": citations_count,
                "response_time_ms": response_time_ms
            }
        else:
            error_message = "未返回有效结果"
            logger.warning(f"⚠️ {matched_platform} {error_message}")
            record_id, _ = save_to_db(
                keyword, matched_platform, prompt, None, 
                prompt_type="api_task", 
                error_message=error_message,
                task_id=task_id,
                task_query_id=task_query_id
            )
            return {
                "keyword": keyword,
                "platform": matched_platform,
                "status": "failed",
                "error_message": error_message,
                "record_id": record_id,
                "citations_count": 0
            }
            
    except Exception as e:
        response_time_ms = int((time.time() - start_time) * 1000)
        error_message = str(e)
        logger.error(f"❌ 执行任务失败: {e}", exc_info=True)
        record_id, _ = save_to_db(
            keyword, matched_platform, prompt, None, 
            prompt_type="api_task", 
            response_time_ms=response_time_ms, 
            error_message=error_message,
            task_id=task_id,
            task_query_id=task_query_id
        )
        return {
            "keyword": keyword,
            "platform": matched_platform,
            "status": "failed",
            "error_message": error_message,
            "record_id": record_id,
            "citations_count": 0,
            "response_time_ms": response_time_ms
        }


def execute_task_job(task_id: int, keywords: List[str], platforms: List[str], query_count: int, settings: Dict[str, Any]):
    """
    在后台线程中执行任务作业
    
    Args:
        task_id: 任务ID
        keywords: 关键词列表
        platforms: 平台列表
        query_count: 查询次数（执行轮数）
        settings: 设置字典
    """
    def run():
        try:
            # 更新任务状态为 pending
            with get_db_connection() as conn:
                cur = conn.cursor()
                cur.execute(
                    "UPDATE task_jobs SET status = 'pending' WHERE id = %s",
                    (task_id,)
                )
                conn.commit()
            
            # 获取 task_query_id 映射（keyword -> task_query_id）
            task_query_map = {}
            with get_db_connection() as conn:
                cur = conn.cursor()
                for keyword in keywords:
                    cur.execute("""
                        SELECT id FROM task_query 
                        WHERE task_id = %s AND query = %s
                        LIMIT 1
                    """, (task_id, keyword))
                    row = cur.fetchone()
                    if row:
                        task_query_map[keyword] = row[0]
            
            results = []
            delay = settings.get("delay_between_tasks", 5)
            
            # 按执行次数循环：外层循环执行次数，中层循环查询条件，内层循环平台
            for round_num in range(1, query_count + 1):
                logger.info(f"🔄 开始第 {round_num}/{query_count} 轮执行")
                
                for keyword in keywords:
                    prompt = keyword  # 使用关键词作为提示词
                    task_query_id = task_query_map.get(keyword)
                    
                    for platform in platforms:
                        result = execute_single_task(keyword, platform, prompt, settings, task_id, task_query_id)
                        results.append(result)
                        
                        # 任务间延迟
                        if delay > 0:
                            time.sleep(delay)
            
            # 更新任务结果
            import json
            with get_db_connection() as conn:
                cur = conn.cursor()
                cur.execute("""
                    UPDATE task_jobs 
                    SET status = 'done', result_data = %s 
                    WHERE id = %s
                """, (json.dumps(results), task_id))
                conn.commit()
            
            logger.info(f"✅ 任务 {task_id} 执行完成")
            
        except Exception as e:
            logger.error(f"❌ 任务 {task_id} 执行失败: {e}", exc_info=True)
            # 更新任务状态为 done，但记录错误
            try:
                import json
                with get_db_connection() as conn:
                    cur = conn.cursor()
                    error_result = [{"error": str(e)}]
                    cur.execute("""
                        UPDATE task_jobs 
                        SET status = 'done', result_data = %s 
                        WHERE id = %s
                    """, (json.dumps(error_result), task_id))
                    conn.commit()
            except:
                pass
    
    # 在后台线程中执行
    thread = threading.Thread(target=run, daemon=True)
    thread.start()

