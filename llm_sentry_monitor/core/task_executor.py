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

logger = logging.getLogger(__name__)


def save_to_db(keyword, platform, prompt, result, prompt_type="default", response_time_ms=None, error_message=None):
    """保存搜索结果到数据库（从 main.py 复用）"""
    try:
        with get_db_connection() as conn:
            cur = conn.cursor()
            
            # 确定搜索状态
            search_status = 'completed' if result and result.get("full_text") else 'failed'
            if error_message:
                search_status = 'failed'
            
            # 1. 插入搜索记录
            cur.execute("""
                INSERT INTO search_records 
                (keyword, platform, prompt_type, prompt, full_answer, response_time_ms, search_status, error_message) 
                VALUES (%s, %s, %s, %s, %s, %s, %s, %s) 
                RETURNING id
            """, (
                keyword, 
                platform, 
                prompt_type, 
                prompt,
                result.get("full_text", "") if result else "",
                response_time_ms,
                search_status,
                error_message
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
            
            # 3. 插入引用 (利用唯一约束自动去重)
            citations_count = 0
            for cite in result.get("citations", []):
                url = cite.get("url", "")
                if not url:
                    continue
                    
                domain = extract_domain(url)
                try:
                    cur.execute("""
                        INSERT INTO citations 
                        (record_id, cite_index, url, domain, title, snippet, site_name) 
                        VALUES (%s, %s, %s, %s, %s, %s, %s)
                        ON CONFLICT (record_id, url) DO NOTHING
                    """, (
                        record_id, 
                        cite.get("cite_index", 0), 
                        url, 
                        domain, 
                        cite.get("title", ""), 
                        cite.get("snippet", ""), 
                        cite.get("site_name", "")
                    ))
                    
                    if cur.rowcount > 0:
                        citations_count += 1
                        # 更新域名统计
                        update_domain_stats(conn, domain, platform)
                        
                except Exception as e:
                    logger.debug(f"插入引用失败（可能重复）: {e}")
            
            logger.info(f"✅ 成功保存 {platform} 的数据，记录 ID: {record_id}")
            logger.info(f"  - 拓展词: {len(result.get('queries', []))} 个")
            logger.info(f"  - 参考网页: {citations_count} 个")
            if response_time_ms:
                logger.info(f"  - 响应时间: {response_time_ms/1000:.2f} 秒")
            
            return record_id, citations_count
                
    except Exception as e:
        logger.error(f"❌ 保存到数据库失败: {e}", exc_info=True)
        return None, 0


def execute_single_task(keyword: str, platform: str, prompt: str, settings: Dict[str, Any]) -> Dict[str, Any]:
    """
    执行单个关键词-平台组合的搜索任务
    
    Args:
        keyword: 搜索关键词
        platform: 平台名称 (deepseek, doubao)
        prompt: 提示词
        settings: 设置字典 (headless, timeout等)
    
    Returns:
        包含执行结果的字典
    """
    headless = settings.get("headless", False)
    timeout = settings.get("timeout", 60000)
    
    providers = {
        "deepseek": DeepSeekWebProvider(headless=headless, timeout=timeout),
        "doubao": DoubaoWebProvider(headless=headless, timeout=timeout)
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
                response_time_ms=response_time_ms
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
                error_message=error_message
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
            error_message=error_message
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


def execute_task_job(task_id: int, keywords: List[str], platforms: List[str], settings: Dict[str, Any]):
    """
    在后台线程中执行任务作业
    
    Args:
        task_id: 任务ID
        keywords: 关键词列表
        platforms: 平台列表
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
            
            results = []
            delay = settings.get("delay_between_tasks", 5)
            
            # 执行所有关键词-平台组合
            for keyword in keywords:
                prompt = keyword  # 使用关键词作为提示词
                for platform in platforms:
                    result = execute_single_task(keyword, platform, prompt, settings)
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

