"""
Status service - Business logic for task status queries
"""
import json
import logging
from typing import List, Dict, Any, Optional
import psycopg2.errors
from core.db import get_db_connection
from utils.encoding import ensure_utf8_string

logger = logging.getLogger(__name__)


def get_doubao_query_tokens(results_by_platform):
    """
    从 results_by_platform 中获取豆包的 query_tokens 合集
    返回逗号分隔的字符串
    """
    doubao_data = results_by_platform.get("doubao") or results_by_platform.get("豆包")
    if not doubao_data:
        return None
    
    query_tokens = doubao_data.get("query_tokens", [])
    if not query_tokens:
        return None
    
    # 提取所有 query，用逗号连接
    queries = [token.get("query", "") for token in query_tokens if token.get("query")]
    return ", ".join(queries) if queries else None


def _parse_json_field(field_value):
    """解析可能是JSON字符串或已解析对象的字段"""
    if isinstance(field_value, (list, dict)):
        return field_value
    return json.loads(field_value) if field_value else None


def _ensure_utf8_list(items):
    """确保列表中的字符串都是UTF-8编码"""
    if not isinstance(items, list):
        return items
    return [ensure_utf8_string(item) if isinstance(item, str) else item for item in items]


def _get_platform_progress(cur, task_id, task_query_ids, platforms, query_count):
    """计算任务的平台进度"""
    num_keywords = len(task_query_ids)
    total_rounds = num_keywords * len(platforms) * query_count if num_keywords > 0 and platforms and query_count else 0
    
    completed_rounds = 0
    failed_rounds = 0
    
    if task_query_ids and platforms:
        placeholders = ','.join(['%s'] * len(task_query_ids))
        cur.execute(f"""
            SELECT search_status, COUNT(*) as count
            FROM search_records
            WHERE task_id = %s 
              AND task_query_id IN ({placeholders})
              AND platform IN ({','.join(['%s'] * len(platforms))})
              AND prompt_type = 'api_task'
            GROUP BY search_status
        """, [task_id] + task_query_ids + [p.lower() for p in platforms])
        
        status_rows = cur.fetchall()
        for status_row in status_rows:
            search_status = status_row[0]
            count = status_row[1]
            if search_status == 'completed':
                completed_rounds = count
            elif search_status == 'failed':
                failed_rounds = count
    
    pending_rounds = max(0, total_rounds - completed_rounds - failed_rounds)
    
    return {
        "completed": completed_rounds,
        "failed": failed_rounds,
        "pending": pending_rounds,
        "total": total_rounds
    }


def _build_round_map(cur, task_id, task_query_ids, platforms):
    """构建轮次映射"""
    round_map = {}
    
    for task_query_id in task_query_ids:
        for platform in platforms:
            platform_lower = platform.lower()
            cur.execute("""
                SELECT id, created_at
                FROM search_records
                WHERE task_id = %s 
                  AND task_query_id = %s 
                  AND platform = %s
                ORDER BY created_at ASC
            """, (task_id, task_query_id, platform_lower))
            record_rows = cur.fetchall()
            
            for round_num, record_row in enumerate(record_rows, start=1):
                record_id = record_row[0]
                key = (task_query_id, platform_lower)
                if key not in round_map:
                    round_map[key] = {}
                round_map[key][record_id] = round_num
    
    return round_map


def _build_results_by_platform(cur, task_id, task_query_ids, platforms, platform_status_map):
    """构建按平台分组的结果"""
    results_by_platform = {}
    
    for platform in platforms:
        platform_lower = platform.lower()
        platform_query_tokens = []
        
        if task_query_ids:
            placeholders = ','.join(['%s'] * len(task_query_ids))
            cur.execute(f"""
                SELECT DISTINCT sq.query, sq.record_id, sq.query_order, sq.id
                FROM search_queries sq
                INNER JOIN search_records sr ON sq.record_id = sr.id
                WHERE sr.task_id = %s 
                  AND sr.task_query_id IN ({placeholders})
                  AND sr.platform = %s
                  AND sr.prompt_type = 'api_task'
                ORDER BY sq.query_order, sq.id
            """, [task_id] + task_query_ids + [platform_lower])
            query_rows = cur.fetchall()
            
            for row in query_rows:
                query = row[0]
                record_id = row[1]
                if query:
                    query = ensure_utf8_string(query)
                    
                    cur.execute("""
                        SELECT url, title, snippet, site_name, cite_index, domain
                        FROM citations
                        WHERE record_id = %s
                        ORDER BY cite_index, id
                    """, (record_id,))
                    citation_rows = cur.fetchall()
                    
                    citations = []
                    for cite_row in citation_rows:
                        citations.append({
                            "url": ensure_utf8_string(cite_row[0] or ""),
                            "title": ensure_utf8_string(cite_row[1] or ""),
                            "snippet": ensure_utf8_string(cite_row[2] or ""),
                            "site_name": ensure_utf8_string(cite_row[3] or ""),
                            "cite_index": cite_row[4] or 0,
                            "domain": ensure_utf8_string(cite_row[5] or "")
                        })
                    
                    platform_query_tokens.append({
                        "query": ensure_utf8_string(query),
                        "citations": citations
                    })
        
        platform_info = platform_status_map.get(platform_lower, {})
        results_by_platform[platform_lower] = {
            "query_tokens": platform_query_tokens,
            "status": platform_info.get("status", "pending"),
            "record_id": platform_info.get("record_id"),
            "citations_count": platform_info.get("citations_count", 0),
            "response_time_ms": platform_info.get("response_time_ms"),
            "error_message": platform_info.get("error_message")
        }
    
    return results_by_platform


def _build_summary_table(cur, task_id, sub_query_logs, task_query_map, platforms, results_by_platform):
    """构建汇总表格数据"""
    summary_table = {}
    
    for sql in sub_query_logs:
        url = sql[3]
        task_query_id = sql[1]
        query = task_query_map.get(task_query_id, "")
        sub_query = ensure_utf8_string(sql[2]) if sql[2] and isinstance(sql[2], str) else (sql[2] or "")
        
        if not sub_query and not url:
            continue
        
        record_id = sql[10]
        citation_id = sql[11] if len(sql) > 11 else None
        
        platform = ""
        if record_id:
            cur.execute("""
                SELECT platform FROM search_records WHERE id = %s LIMIT 1
            """, (record_id,))
            platform_row = cur.fetchone()
            if platform_row:
                platform = ensure_utf8_string(platform_row[0]) if isinstance(platform_row[0], str) else platform_row[0]
        
        if not platform and task_query_id:
            cur.execute("""
                SELECT DISTINCT platform FROM search_records 
                WHERE task_id = %s AND task_query_id = %s LIMIT 1
            """, (task_id, task_query_id))
            platform_row = cur.fetchone()
            if platform_row:
                platform = ensure_utf8_string(platform_row[0]) if isinstance(platform_row[0], str) else platform_row[0]
        
        if not platform and not record_id and sub_query and task_query_id:
            cur.execute("""
                SELECT DISTINCT platform FROM search_records 
                WHERE task_id = %s AND task_query_id = %s
                ORDER BY platform
            """, (task_id, task_query_id))
            platform_rows = cur.fetchall()
            if platform_rows:
                platform = ensure_utf8_string(platform_rows[0][0]) if isinstance(platform_rows[0][0], str) else platform_rows[0][0]
            else:
                for platform_name in platforms:
                    platform_key = ensure_utf8_string(platform_name) if isinstance(platform_name, str) else platform_name
                    key = (query, platform_key, sub_query)
                    if key not in summary_table:
                        summary_table[key] = set()
                continue
        
        platform_lower = platform.lower() if platform else ""
        if (platform_lower == "doubao" or platform_lower == "豆包") and not sub_query:
            doubao_queries = get_doubao_query_tokens(results_by_platform)
            if doubao_queries:
                sub_query = doubao_queries
        
        key = (query, platform, sub_query)
        if key not in summary_table:
            summary_table[key] = set()
        
        if url:
            if citation_id:
                summary_table[key].add(citation_id)
            else:
                summary_table[key].add(url)
    
    return [
        {
            "query": query,
            "platform": platform,
            "sub_query": sub_query,
            "count": len(citation_ids)
        }
        for (query, platform, sub_query), citation_ids in summary_table.items()
    ]


def _build_detail_logs(cur, task_id, sub_query_logs, task_query_map, round_map, results_by_platform):
    """构建详细日志数据"""
    detail_logs = []
    
    for sql in sub_query_logs:
        url = sql[3]
        task_query_id = sql[1]
        query = task_query_map.get(task_query_id, "")
        sub_query = ensure_utf8_string(sql[2]) if sql[2] and isinstance(sql[2], str) else (sql[2] or "")
        
        if not sub_query and not url:
            continue
        
        url = ensure_utf8_string(url) if url and isinstance(url, str) else (url or "")
        domain = ensure_utf8_string(sql[4]) if sql[4] and isinstance(sql[4], str) else (sql[4] or "")
        title = ensure_utf8_string(sql[5]) if sql[5] and isinstance(sql[5], str) else (sql[5] or "")
        snippet = ensure_utf8_string(sql[6]) if sql[6] and isinstance(sql[6], str) else (sql[6] or "")
        created_at = sql[9]
        record_id = sql[10]
        
        platform = ""
        round_num = None
        if record_id:
            cur.execute("""
                SELECT platform FROM search_records WHERE id = %s LIMIT 1
            """, (record_id,))
            platform_row = cur.fetchone()
            if platform_row:
                platform = ensure_utf8_string(platform_row[0]) if isinstance(platform_row[0], str) else platform_row[0]
                
                key = (task_query_id, platform.lower())
                if key in round_map and record_id in round_map[key]:
                    round_num = round_map[key][record_id]
        
        platform_lower = platform.lower() if platform else ""
        if (platform_lower == "doubao" or platform_lower == "豆包") and not sub_query:
            doubao_queries = get_doubao_query_tokens(results_by_platform)
            if doubao_queries:
                sub_query = doubao_queries
        
        detail_logs.append({
            "task_id": task_id,
            "query": query,
            "round": round_num,
            "platform": platform,
            "sub_query": sub_query,
            "time": created_at.isoformat() if created_at else None,
            "domain": domain,
            "url": url,
            "title": title,
            "snippet": snippet
        })
    
    return detail_logs


def get_task_status_data(task_ids: List[int]) -> Dict[str, Any]:
    """
    查询任务状态数据
    
    Args:
        task_ids: 任务ID列表
    
    Returns:
        包含status和data的字典
    
    Raises:
        ValueError: 参数验证失败
        Exception: 数据库查询失败
    """
    if not task_ids:
        raise ValueError("任务ID列表不能为空")
    
    try:
        with get_db_connection() as conn:
            cur = conn.cursor()
            
            # 单个任务查询
            if len(task_ids) == 1:
                return _get_single_task_status(cur, task_ids[0])
            else:
                return _get_multiple_tasks_status(cur, task_ids)
            
    except psycopg2.errors.UndefinedTable as e:
        error_msg = str(e)
        if "task_jobs" in error_msg:
            logger.warning(f"数据库表 task_jobs 不存在: {e}")
            return {
                "status": "none",
                "data": {
                    "error": True,
                    "error_type": "table_not_found",
                    "message": "数据库表 task_jobs 不存在，请检查数据库迁移状态",
                    "detail": "请运行数据库迁移脚本以创建必要的表结构"
                }
            }
        else:
            logger.error(f"数据库表不存在: {e}", exc_info=True)
            return {
                "status": "none",
                "data": {
                    "error": True,
                    "error_type": "table_not_found",
                    "message": f"数据库表不存在: {str(e)}",
                    "detail": "请检查数据库迁移状态"
                }
            }
    except Exception as e:
        logger.error(f"查询任务状态失败: {e}", exc_info=True)
        raise


def _get_single_task_status(cur, task_id: int) -> Dict[str, Any]:
    """查询单个任务的详细状态"""
    cur.execute("""
        SELECT id, keywords, platforms, query_count, status, result_data, created_at, updated_at
        FROM task_jobs
        WHERE id = %s
    """, (task_id,))
    
    row = cur.fetchone()
    
    if not row:
        return {"status": "none", "data": None}
    
    task_id, keywords_json, platforms_json, query_count, status, result_data_json, created_at, updated_at = row
    
    # 解析JSON字段
    keywords = _parse_json_field(keywords_json) or []
    keywords = _ensure_utf8_list(keywords)
    
    platforms = _parse_json_field(platforms_json) or []
    platforms = _ensure_utf8_list(platforms)
    
    result_data = _parse_json_field(result_data_json) or {}
    
    # 查询task_query数据
    cur.execute("""
        SELECT id, query, created_at
        FROM task_query
        WHERE task_id = %s
        ORDER BY id
    """, (task_id,))
    task_queries = cur.fetchall()
    
    # 查询executor_sub_query_log数据
    task_query_ids = [tq[0] for tq in task_queries]
    sub_query_logs = []
    if task_query_ids:
        placeholders = ','.join(['%s'] * len(task_query_ids))
        cur.execute(f"""
            SELECT id, task_query_id, sub_query, url, domain, title, snippet, site_name, cite_index, created_at, record_id, citation_id
            FROM executor_sub_query_log
            WHERE task_query_id IN ({placeholders})
            ORDER BY task_query_id, created_at
        """, task_query_ids)
        sub_query_logs = cur.fetchall()
    
    # 构建基础响应数据
    response_data = {
        "task_id": task_id,
        "keywords": keywords,
        "platforms": platforms,
        "query_count": query_count,
        "created_at": created_at.isoformat() if created_at else None,
        "updated_at": updated_at.isoformat() if updated_at else None,
        "task_queries": [
            {
                "id": tq[0],
                "query": ensure_utf8_string(tq[1]) if isinstance(tq[1], str) else tq[1],
                "created_at": tq[2].isoformat() if tq[2] else None
            }
            for tq in task_queries
        ],
        "sub_query_logs": [
            {
                "id": sql[0],
                "task_query_id": sql[1],
                "sub_query": ensure_utf8_string(sql[2]) if sql[2] and isinstance(sql[2], str) else sql[2],
                "url": ensure_utf8_string(sql[3]) if sql[3] and isinstance(sql[3], str) else sql[3],
                "domain": ensure_utf8_string(sql[4]) if sql[4] and isinstance(sql[4], str) else sql[4],
                "title": ensure_utf8_string(sql[5]) if sql[5] and isinstance(sql[5], str) else sql[5],
                "snippet": ensure_utf8_string(sql[6]) if sql[6] and isinstance(sql[6], str) else sql[6],
                "site_name": ensure_utf8_string(sql[7]) if sql[7] and isinstance(sql[7], str) else sql[7],
                "cite_index": sql[8],
                "created_at": sql[9].isoformat() if sql[9] else None
            }
            for sql in sub_query_logs
        ]
    }
    
    # 构建task_query映射
    task_query_map = {}
    for tq in task_queries:
        task_query_map[tq[0]] = ensure_utf8_string(tq[1]) if isinstance(tq[1], str) else tq[1]
    
    # 推断轮次信息
    round_map = _build_round_map(cur, task_id, task_query_ids, platforms) if task_query_ids and platforms else {}
    
    # 从result_data提取平台状态
    platform_status_map = {}
    if result_data and isinstance(result_data, list):
        for result_item in result_data:
            if isinstance(result_item, dict):
                platform = result_item.get("platform", "").lower()
                platform_status_map[platform] = {
                    "status": result_item.get("status", "pending"),
                    "record_id": result_item.get("record_id"),
                    "citations_count": result_item.get("citations_count", 0),
                    "response_time_ms": result_item.get("response_time_ms"),
                    "error_message": result_item.get("error_message")
                }
    
    # 构建results_by_platform
    results_by_platform = _build_results_by_platform(
        cur, task_id, task_query_ids, platforms, platform_status_map
    ) if keywords and platforms else {}
    
    # 计算平台进度
    platform_progress = _get_platform_progress(
        cur, task_id, task_query_ids, platforms, query_count
    ) if task_query_ids and platforms else {}
    
    # 构建汇总表格
    summary_table = _build_summary_table(
        cur, task_id, sub_query_logs, task_query_map, platforms, results_by_platform
    )
    response_data["summary_table"] = summary_table
    
    # 构建详细日志
    detail_logs = _build_detail_logs(
        cur, task_id, sub_query_logs, task_query_map, round_map, results_by_platform
    )
    response_data["detail_logs"] = detail_logs
    
    # 添加其他数据
    if status == "done" and result_data:
        response_data["results"] = result_data
    
    if results_by_platform:
        response_data["results_by_platform"] = results_by_platform
        response_data["platform_progress"] = platform_progress
    
    return {"status": status, "data": response_data}


def _get_multiple_tasks_status(cur, task_ids: List[int]) -> Dict[str, Any]:
    """查询多个任务的状态（简化版）"""
    placeholders = ','.join(['%s'] * len(task_ids))
    cur.execute(f"""
        SELECT id, keywords, platforms, query_count, status, result_data, created_at, updated_at
        FROM task_jobs
        WHERE id IN ({placeholders})
        ORDER BY id
    """, task_ids)
    
    rows = cur.fetchall()
    
    if not rows:
        return {"status": "none", "data": None}
    
    tasks_data = []
    for row in rows:
        task_id, keywords_json, platforms_json, query_count, status, result_data_json, created_at, updated_at = row
        
        keywords = _ensure_utf8_list(_parse_json_field(keywords_json) or [])
        platforms = _ensure_utf8_list(_parse_json_field(platforms_json) or [])
        
        # 查询task_query数据
        cur.execute("""
            SELECT id, query, created_at
            FROM task_query
            WHERE task_id = %s
            ORDER BY id
        """, (task_id,))
        task_queries = cur.fetchall()
        
        task_query_map = {}
        task_query_list = []
        for tq in task_queries:
            task_query_id = tq[0]
            query_text = ensure_utf8_string(tq[1]) if isinstance(tq[1], str) else tq[1]
            task_query_map[task_query_id] = query_text
            task_query_list.append({
                "id": task_query_id,
                "query": query_text,
                "created_at": tq[2].isoformat() if tq[2] else None
            })
        
        # 查询executor_sub_query_log数据
        task_query_ids = [tq[0] for tq in task_queries]
        sub_query_logs = []
        if task_query_ids:
            placeholders_sql = ','.join(['%s'] * len(task_query_ids))
            cur.execute(f"""
                SELECT esql.id, esql.task_query_id, esql.sub_query, esql.url, esql.domain, 
                       esql.title, esql.snippet, esql.site_name, esql.cite_index, esql.created_at,
                       esql.record_id, esql.citation_id
                FROM executor_sub_query_log esql
                WHERE esql.task_query_id IN ({placeholders_sql})
                ORDER BY esql.task_query_id, esql.created_at
            """, task_query_ids)
            sub_query_logs = cur.fetchall()
        
        # 构建轮次映射
        round_map = _build_round_map(cur, task_id, task_query_ids, platforms) if task_query_ids and platforms else {}
        
        # 构建results_by_platform（简化版）
        task_results_by_platform = {}
        if keywords and platforms:
            for platform in platforms:
                platform_lower = platform.lower()
                platform_query_tokens = []
                
                if task_query_ids:
                    placeholders_query = ','.join(['%s'] * len(task_query_ids))
                    cur.execute(f"""
                        SELECT DISTINCT sq.query, sq.record_id, sq.query_order, sq.id
                        FROM search_queries sq
                        INNER JOIN search_records sr ON sq.record_id = sr.id
                        WHERE sr.task_id = %s 
                          AND sr.task_query_id IN ({placeholders_query})
                          AND sr.platform = %s
                          AND sr.prompt_type = 'api_task'
                        ORDER BY sq.query_order, sq.id
                    """, [task_id] + task_query_ids + [platform_lower])
                    query_rows = cur.fetchall()
                    
                    for row in query_rows:
                        query = row[0]
                        if query:
                            query = ensure_utf8_string(query)
                            platform_query_tokens.append({
                                "query": query,
                                "citations": []
                            })
                
                task_results_by_platform[platform_lower] = {
                    "query_tokens": platform_query_tokens
                }
        
        # 构建汇总表格
        summary_table = _build_summary_table(
            cur, task_id, sub_query_logs, task_query_map, platforms, task_results_by_platform
        )
        
        # 构建详细日志
        detail_logs = _build_detail_logs(
            cur, task_id, sub_query_logs, task_query_map, round_map, task_results_by_platform
        )
        
        tasks_data.append({
            "task_id": task_id,
            "keywords": keywords,
            "platforms": platforms,
            "query_count": query_count,
            "status": status,
            "created_at": created_at.isoformat() if created_at else None,
            "updated_at": updated_at.isoformat() if updated_at else None,
            "task_queries": task_query_list,
            "summary_table": summary_table,
            "detail_logs": detail_logs
        })
    
    return {"status": "multiple", "data": {"tasks": tasks_data}}
