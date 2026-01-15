"""
api.py - FastAPI 应用
提供 REST API 接口用于任务管理和状态查询
"""
import os
import json
import logging
from typing import List, Dict, Any, Optional
from fastapi import FastAPI, HTTPException, Query
from fastapi.staticfiles import StaticFiles
from fastapi.responses import FileResponse, Response
from pydantic import BaseModel
import csv
import io
from core.db import get_db_connection
from core.task_executor import execute_task_job
from providers.bocha_api import BochaApiProvider
from providers.doubao_web import ensure_utf8_string

# 配置日志
logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(name)s - %(levelname)s - %(message)s'
)
logger = logging.getLogger(__name__)

app = FastAPI(
    title="LLM Sentry Monitor API",
    description="GEO 品牌曝光监测系统 - 任务管理 API",
    version="1.0.0"
)


# 请求模型
class MockRequest(BaseModel):
    keywords: List[str]
    platforms: List[str] = ["deepseek"]
    query_count: int = 1  # 查询次数（执行轮数），默认1次
    settings: Optional[Dict[str, Any]] = None


# 响应模型
class MockResponse(BaseModel):
    task_id: int


class StatusResponse(BaseModel):
    status: str  # none, pending, done
    data: Optional[Dict[str, Any]] = None


@app.post("/mock", response_model=MockResponse)
async def create_task(request: MockRequest):
    """
    创建新的搜索任务
    
    - **keywords**: 搜索关键词列表
    - **platforms**: 平台列表 (deepseek, doubao)
    - **query_count**: 查询次数（执行轮数），默认1次
    - **settings**: 可选设置 (headless, timeout, delay_between_tasks)
    """
    try:
        # 验证输入
        if not request.keywords:
            raise HTTPException(status_code=400, detail="keywords 不能为空")
        
        if not request.platforms:
            raise HTTPException(status_code=400, detail="platforms 不能为空")
        
        if request.query_count < 1:
            raise HTTPException(status_code=400, detail="query_count 必须大于0")
        
        # 加载默认设置
        default_settings = {
            "headless": False,
            "timeout": 60000,
            "delay_between_tasks": 5
        }
        
        # 合并用户设置
        settings = {**default_settings, **(request.settings or {})}
        
        # 创建任务记录
        with get_db_connection() as conn:
            cur = conn.cursor()
            cur.execute("""
                INSERT INTO task_jobs (keywords, platforms, query_count, status, settings)
                VALUES (%s, %s, %s, 'pending', %s)
                RETURNING id
            """, (
                json.dumps(request.keywords),
                json.dumps(request.platforms),
                request.query_count,
                json.dumps(settings)
            ))
            task_id = cur.fetchone()[0]
            
            # 为每个查询条件创建 task_query 记录
            for keyword in request.keywords:
                cur.execute("""
                    INSERT INTO task_query (task_id, query)
                    VALUES (%s, %s)
                """, (task_id, keyword))
            
            conn.commit()
        
        logger.info(f"创建任务 {task_id}: keywords={request.keywords}, platforms={request.platforms}, query_count={request.query_count}")
        
        # 启动后台任务
        execute_task_job(task_id, request.keywords, request.platforms, request.query_count, settings)
        
        return MockResponse(task_id=task_id)
        
    except HTTPException:
        raise
    except Exception as e:
        logger.error(f"创建任务失败: {e}", exc_info=True)
        raise HTTPException(status_code=500, detail=f"创建任务失败: {str(e)}")


@app.get("/status", response_model=StatusResponse)
async def get_task_status(
    id: Optional[int] = Query(None, description="单个任务ID"),
    ids: Optional[str] = Query(None, description="多个任务ID，逗号分隔")
):
    """
    查询任务状态
    
    - **id**: 单个任务ID
    - **ids**: 多个任务ID（逗号分隔），与 id 参数二选一
    
    返回:
    - **status**: 任务状态 (none, pending, done)
    - **data**: 任务数据（当 status != none 时）
    """
    try:
        # 确定要查询的任务ID列表
        task_ids = []
        if ids:
            # 解析逗号分隔的ID列表
            task_ids = [int(tid.strip()) for tid in ids.split(',') if tid.strip()]
        elif id:
            task_ids = [id]
        else:
            raise HTTPException(status_code=400, detail="必须提供 id 或 ids 参数")
        
        if not task_ids:
            raise HTTPException(status_code=400, detail="任务ID列表不能为空")
        
        with get_db_connection() as conn:
            cur = conn.cursor()
            
            # 如果只有一个任务ID，返回单个任务数据（保持向后兼容）
            if len(task_ids) == 1:
                task_id = task_ids[0]
                cur.execute("""
                    SELECT id, keywords, platforms, query_count, status, result_data, created_at, updated_at
                    FROM task_jobs
                    WHERE id = %s
                """, (task_id,))
                
                row = cur.fetchone()
                
                if not row:
                    return StatusResponse(status="none", data=None)
                
                task_id, keywords_json, platforms_json, query_count, status, result_data_json, created_at, updated_at = row
                
                # 解析 JSON 数据
                if isinstance(keywords_json, (list, dict)):
                    keywords = keywords_json
                else:
                    keywords = json.loads(keywords_json) if keywords_json else []
                
                if isinstance(keywords, list):
                    keywords = [ensure_utf8_string(k) if isinstance(k, str) else k for k in keywords]
                
                if isinstance(platforms_json, (list, dict)):
                    platforms = platforms_json
                else:
                    platforms = json.loads(platforms_json) if platforms_json else []
                
                if isinstance(platforms, list):
                    platforms = [ensure_utf8_string(p) if isinstance(p, str) else p for p in platforms]
                
                if isinstance(result_data_json, (list, dict)):
                    result_data = result_data_json
                else:
                    result_data = json.loads(result_data_json) if result_data_json else {}
                
                # 查询 task_query 数据
                cur.execute("""
                    SELECT id, query, created_at
                    FROM task_query
                    WHERE task_id = %s
                    ORDER BY id
                """, (task_id,))
                task_queries = cur.fetchall()
                
                # 查询 executor_sub_query_log 数据（包含 record_id 用于推断轮次）
                task_query_ids = [tq[0] for tq in task_queries]
                sub_query_logs = []
                if task_query_ids:
                    placeholders = ','.join(['%s'] * len(task_query_ids))
                    cur.execute(f"""
                        SELECT id, task_query_id, sub_query, url, domain, title, snippet, site_name, cite_index, created_at, record_id
                        FROM executor_sub_query_log
                        WHERE task_query_id IN ({placeholders})
                        ORDER BY task_query_id, created_at
                    """, task_query_ids)
                    sub_query_logs = cur.fetchall()
                
                # 构建响应数据
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
                
                # 构建 task_query_id 到 query 的映射（用于汇总表格和详细日志）
                task_query_map = {}
                for tq in task_queries:
                    task_query_map[tq[0]] = ensure_utf8_string(tq[1]) if isinstance(tq[1], str) else tq[1]
                
                # 推断轮次信息
                round_map = {}
                if task_query_ids and platforms:
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
                
                # 构建汇总表格数据
                # 使用字典存储 (query, platform, sub_query) -> set(record_id) 来统计查询次数
                summary_table = {}
                for sql in sub_query_logs:
                    # 过滤：只处理有 url 的记录（过滤掉 A 类型，取消单独保存的 sub_query）
                    url = sql[3]  # url 字段
                    if not url:
                        continue  # 跳过没有 url 的记录
                    
                    task_query_id = sql[1]
                    query = task_query_map.get(task_query_id, "")
                    sub_query = ensure_utf8_string(sql[2]) if sql[2] and isinstance(sql[2], str) else (sql[2] or "")
                    record_id = sql[10]  # record_id 字段
                    
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
                    
                    key = (query, platform, sub_query)
                    if key not in summary_table:
                        summary_table[key] = set()
                    if record_id:
                        summary_table[key].add(record_id)
                
                response_data["summary_table"] = [
                    {
                        "query": query,
                        "platform": platform,
                        "sub_query": sub_query,
                        "count": len(record_ids)  # 统计不同的record_id数量（查询次数）
                    }
                    for (query, platform, sub_query), record_ids in summary_table.items()
                ]
                
                # 构建详细日志数据
                detail_logs = []
                for sql in sub_query_logs:
                    # 过滤：只处理有 url 的记录（过滤掉 A 类型，取消单独保存的 sub_query）
                    url = sql[3]  # url 字段
                    if not url:
                        continue  # 跳过没有 url 的记录
                    
                    task_query_id = sql[1]
                    query = task_query_map.get(task_query_id, "")
                    sub_query = ensure_utf8_string(sql[2]) if sql[2] and isinstance(sql[2], str) else (sql[2] or "")
                    url = ensure_utf8_string(url) if isinstance(url, str) else url
                    domain = ensure_utf8_string(sql[4]) if sql[4] and isinstance(sql[4], str) else (sql[4] or "")
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
                    
                    detail_logs.append({
                        "task_id": task_id,
                        "query": query,
                        "round": round_num,
                        "platform": platform,
                        "sub_query": sub_query,
                        "time": created_at.isoformat() if created_at else None,
                        "domain": domain,
                        "url": url
                    })
                
                response_data["detail_logs"] = detail_logs
                
                # 如果任务完成，添加结果
                if status == "done" and result_data:
                    response_data["results"] = result_data
                
                # 查询内部查询的分词（保持向后兼容）
                query_tokens = []
                results_by_platform = {}
                
                # 计算总轮次数：关键词数 × 平台数 × 查询次数
                # 每个 (keyword, platform) 组合会执行 query_count 轮
                # 使用 task_query_ids 的长度（如果为空则使用 keywords 的长度作为备选）
                num_keywords = len(task_query_ids) if task_query_ids else len(keywords) if keywords else 0
                total_rounds = num_keywords * len(platforms) * query_count if num_keywords > 0 and platforms and query_count else 0
                
                # 统计实际轮次完成情况：通过查询 search_records 表
                completed_rounds = 0
                failed_rounds = 0
                
                if task_query_ids and platforms:
                    # 查询所有 search_records，按 search_status 分组统计
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
                
                # 计算待处理轮次数
                pending_rounds = max(0, total_rounds - completed_rounds - failed_rounds)
                
                platform_progress = {
                    "completed": completed_rounds,
                    "failed": failed_rounds,
                    "pending": pending_rounds,
                    "total": total_rounds
                }
                
                # 从 result_data 中提取平台执行状态（用于 results_by_platform）
                platform_status_map = {}
                if result_data and isinstance(result_data, list):
                    for result_item in result_data:
                        if isinstance(result_item, dict):
                            platform = result_item.get("platform", "").lower()
                            status = result_item.get("status", "pending")
                            platform_status_map[platform] = {
                                "status": status,
                                "record_id": result_item.get("record_id"),
                                "citations_count": result_item.get("citations_count", 0),
                                "response_time_ms": result_item.get("response_time_ms"),
                                "error_message": result_item.get("error_message")
                            }
                
                if keywords and platforms:
                    for platform in platforms:
                        platform_lower = platform.lower()
                        platform_query_tokens = []
                        query_rows = []
                        
                        # 优先使用新的关联字段查询（通过 task_id 和 task_query_id）
                        task_query_ids = [tq[0] for tq in task_queries]
                        if task_query_ids:
                            placeholders = ','.join(['%s'] * len(task_query_ids))
                            # 使用新的关联字段查询，效率更高
                            query_sql = f"""
                                SELECT DISTINCT sq.query, sq.record_id, sq.query_order, sq.id
                                FROM search_queries sq
                                INNER JOIN search_records sr ON sq.record_id = sr.id
                                WHERE sr.task_id = %s 
                                  AND sr.task_query_id IN ({placeholders})
                                  AND sr.platform = %s
                                  AND sr.prompt_type = 'api_task'
                                ORDER BY sq.query_order, sq.id
                            """
                            cur.execute(query_sql, [task_id] + task_query_ids + [platform_lower])
                            query_rows = cur.fetchall()
                        
                        # 如果没有结果，回退到旧的查询方式（向后兼容）
                        if not query_rows:
                            placeholders_old = []
                            params_old = []
                            for keyword in keywords:
                                placeholders_old.append("(sr.keyword = %s AND sr.platform = %s AND sr.prompt_type = 'api_task')")
                                params_old.extend([keyword, platform_lower])
                            
                            if placeholders_old:
                                query_sql_old = f"""
                                    SELECT DISTINCT sq.query, sq.record_id, sq.query_order, sq.id
                                    FROM search_queries sq
                                    INNER JOIN search_records sr ON sq.record_id = sr.id
                                    WHERE {' OR '.join(placeholders_old)}
                                    ORDER BY sq.query_order, sq.id
                                """
                                cur.execute(query_sql_old, params_old)
                                query_rows = cur.fetchall()
                        
                        # 处理查询结果
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
                        
                        query_tokens.extend(platform_query_tokens)
                
                if query_tokens:
                    response_data["query_tokens"] = query_tokens
                
                if results_by_platform:
                    response_data["results_by_platform"] = results_by_platform
                    response_data["platform_progress"] = platform_progress
                
                return StatusResponse(status=status, data=response_data)
            
            else:
                # 多个任务ID，返回完整任务数据
                placeholders = ','.join(['%s'] * len(task_ids))
                cur.execute(f"""
                    SELECT id, keywords, platforms, query_count, status, result_data, created_at, updated_at
                    FROM task_jobs
                    WHERE id IN ({placeholders})
                    ORDER BY id
                """, task_ids)
                
                rows = cur.fetchall()
                
                if not rows:
                    return StatusResponse(status="none", data=None)
                
                tasks_data = []
                for row in rows:
                    task_id, keywords_json, platforms_json, query_count, status, result_data_json, created_at, updated_at = row
                    
                    if isinstance(keywords_json, (list, dict)):
                        keywords = keywords_json
                    else:
                        keywords = json.loads(keywords_json) if keywords_json else []
                    
                    if isinstance(keywords, list):
                        keywords = [ensure_utf8_string(k) if isinstance(k, str) else k for k in keywords]
                    
                    if isinstance(platforms_json, (list, dict)):
                        platforms = platforms_json
                    else:
                        platforms = json.loads(platforms_json) if platforms_json else []
                    
                    if isinstance(platforms, list):
                        platforms = [ensure_utf8_string(p) if isinstance(p, str) else p for p in platforms]
                    
                    # 查询 task_query 数据
                    cur.execute("""
                        SELECT id, query, created_at
                        FROM task_query
                        WHERE task_id = %s
                        ORDER BY id
                    """, (task_id,))
                    task_queries = cur.fetchall()
                    
                    # 构建 task_query_id 到 query 的映射
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
                    
                    # 查询 executor_sub_query_log 数据
                    task_query_ids = [tq[0] for tq in task_queries]
                    sub_query_logs = []
                    if task_query_ids:
                        placeholders_sql = ','.join(['%s'] * len(task_query_ids))
                        cur.execute(f"""
                            SELECT esql.id, esql.task_query_id, esql.sub_query, esql.url, esql.domain, 
                                   esql.title, esql.snippet, esql.site_name, esql.cite_index, esql.created_at,
                                   esql.record_id
                            FROM executor_sub_query_log esql
                            WHERE esql.task_query_id IN ({placeholders_sql})
                            ORDER BY esql.task_query_id, esql.created_at
                        """, task_query_ids)
                        sub_query_logs = cur.fetchall()
                    
                    # 推断轮次信息：通过 search_records 的 created_at 和任务关联推断
                    # 对于同一个 task_id + task_query_id + platform，按 created_at 排序来确定轮次
                    round_map = {}  # (task_query_id, platform) -> {record_id: round_num}
                    
                    if task_query_ids and platforms:
                        for task_query_id in task_query_ids:
                            for platform in platforms:
                                platform_lower = platform.lower()
                                # 查询该 task_query_id + platform 的所有 search_records，按 created_at 排序
                                cur.execute("""
                                    SELECT id, created_at
                                    FROM search_records
                                    WHERE task_id = %s 
                                      AND task_query_id = %s 
                                      AND platform = %s
                                    ORDER BY created_at ASC
                                """, (task_id, task_query_id, platform_lower))
                                record_rows = cur.fetchall()
                                
                                # 根据排序位置推断轮次（第1个为轮次1，第2个为轮次2，以此类推）
                                for round_num, record_row in enumerate(record_rows, start=1):
                                    record_id = record_row[0]
                                    key = (task_query_id, platform_lower)
                                    if key not in round_map:
                                        round_map[key] = {}
                                    round_map[key][record_id] = round_num
                    
                    # 构建汇总表格数据：查询词、平台、sub_query、sub_query次数
                    # 使用字典存储 (query, platform, sub_query) -> set(record_id) 来统计查询次数
                    summary_table = {}
                    # 使用 (query, platform, sub_query) 作为 key
                    for sql in sub_query_logs:
                        # 过滤：只处理有 url 的记录（过滤掉 A 类型，取消单独保存的 sub_query）
                        url = sql[3]  # url 字段
                        if not url:
                            continue  # 跳过没有 url 的记录
                        
                        task_query_id = sql[1]
                        query = task_query_map.get(task_query_id, "")
                        sub_query = ensure_utf8_string(sql[2]) if sql[2] and isinstance(sql[2], str) else (sql[2] or "")
                        record_id = sql[10]  # record_id 字段
                        
                        # 需要确定平台，通过 record_id 查询 search_records
                        platform = ""
                        if record_id:
                            cur.execute("""
                                SELECT platform FROM search_records WHERE id = %s LIMIT 1
                            """, (record_id,))
                            platform_row = cur.fetchone()
                            if platform_row:
                                platform = ensure_utf8_string(platform_row[0]) if isinstance(platform_row[0], str) else platform_row[0]
                        
                        if not platform:
                            # 如果没有 record_id，尝试从 task_query 关联的 search_records 推断
                            # 这里简化处理，使用第一个匹配的平台
                            if task_query_id:
                                cur.execute("""
                                    SELECT DISTINCT platform FROM search_records 
                                    WHERE task_id = %s AND task_query_id = %s LIMIT 1
                                """, (task_id, task_query_id))
                                platform_row = cur.fetchone()
                                if platform_row:
                                    platform = ensure_utf8_string(platform_row[0]) if isinstance(platform_row[0], str) else platform_row[0]
                        
                        key = (query, platform, sub_query)
                        if key not in summary_table:
                            summary_table[key] = set()
                        if record_id:
                            summary_table[key].add(record_id)
                    
                    # 转换为列表格式
                    summary_table_list = [
                        {
                            "query": query,
                            "platform": platform,
                            "sub_query": sub_query,
                            "count": len(record_ids)  # 统计不同的record_id数量（查询次数）
                        }
                        for (query, platform, sub_query), record_ids in summary_table.items()
                    ]
                    
                    # 构建详细日志数据：task_id、查询词、轮次、平台、sub_query、时间、域名、网址超链
                    detail_logs = []
                    for sql in sub_query_logs:
                        # 过滤：只处理有 url 的记录（过滤掉 A 类型，取消单独保存的 sub_query）
                        url = sql[3]  # url 字段
                        if not url:
                            continue  # 跳过没有 url 的记录
                        
                        task_query_id = sql[1]
                        query = task_query_map.get(task_query_id, "")
                        sub_query = ensure_utf8_string(sql[2]) if sql[2] and isinstance(sql[2], str) else (sql[2] or "")
                        url = ensure_utf8_string(url) if isinstance(url, str) else url
                        domain = ensure_utf8_string(sql[4]) if sql[4] and isinstance(sql[4], str) else (sql[4] or "")
                        created_at = sql[9]
                        record_id = sql[10]
                        
                        # 确定平台和轮次
                        platform = ""
                        round_num = None
                        if record_id:
                            cur.execute("""
                                SELECT platform FROM search_records WHERE id = %s LIMIT 1
                            """, (record_id,))
                            platform_row = cur.fetchone()
                            if platform_row:
                                platform = ensure_utf8_string(platform_row[0]) if isinstance(platform_row[0], str) else platform_row[0]
                                
                                # 推断轮次
                                key = (task_query_id, platform.lower())
                                if key in round_map and record_id in round_map[key]:
                                    round_num = round_map[key][record_id]
                        
                        detail_logs.append({
                            "task_id": task_id,
                            "query": query,
                            "round": round_num,
                            "platform": platform,
                            "sub_query": sub_query,
                            "time": created_at.isoformat() if created_at else None,
                            "domain": domain,
                            "url": url
                        })
                    
                    tasks_data.append({
                        "task_id": task_id,
                        "keywords": keywords,
                        "platforms": platforms,
                        "query_count": query_count,
                        "status": status,
                        "created_at": created_at.isoformat() if created_at else None,
                        "updated_at": updated_at.isoformat() if updated_at else None,
                        "task_queries": task_query_list,
                        "summary_table": summary_table_list,
                        "detail_logs": detail_logs
                    })
                
                return StatusResponse(status="multiple", data={"tasks": tasks_data})
            
    except Exception as e:
        logger.error(f"查询任务状态失败: {e}", exc_info=True)
        raise HTTPException(status_code=500, detail=f"查询任务状态失败: {str(e)}")


# 静态文件服务
import os
static_dir = os.path.join(os.path.dirname(__file__), "static")
if os.path.exists(static_dir):
    app.mount("/static", StaticFiles(directory=static_dir), name="static")

@app.get("/")
async def root():
    """前端页面入口"""
    static_dir_path = os.path.join(os.path.dirname(__file__), "static")
    index_path = os.path.join(static_dir_path, "index.html")
    if os.path.exists(index_path):
        return FileResponse(index_path)
    # 如果没有前端页面，返回API信息
    return {
        "message": "LLM Sentry Monitor API",
        "version": "1.0.0",
        "endpoints": {
            "POST /mock": "创建新的搜索任务",
            "GET /status?id=<task_id>": "查询任务状态",
            "POST /bocha/search?query=<query>": "博查实时搜索"
        }
    }


@app.get("/health")
async def health():
    """健康检查"""
    try:
        # 测试数据库连接
        with get_db_connection() as conn:
            cur = conn.cursor()
            cur.execute("SELECT 1")
            cur.fetchone()
        return {"status": "healthy", "database": "connected"}
    except Exception as e:
        logger.error(f"健康检查失败: {e}")
        return {"status": "unhealthy", "error": str(e)}


@app.get("/export")
async def export_task_data(ids: str = Query(..., description="任务ID列表，逗号分隔")):
    """
    导出任务明细数据（CSV格式）
    
    - **ids**: 任务ID列表，逗号分隔
    
    返回CSV文件，包含：原始query、平台、sub_query、网址、时间
    """
    try:
        # 解析任务ID列表
        task_ids = [int(tid.strip()) for tid in ids.split(',') if tid.strip()]
        
        if not task_ids:
            raise HTTPException(status_code=400, detail="任务ID列表不能为空")
        
        # 查询数据
        with get_db_connection() as conn:
            cur = conn.cursor()
            
            # 查询所有相关的 task_query 和 executor_sub_query_log
            placeholders = ','.join(['%s'] * len(task_ids))
            cur.execute(f"""
                SELECT 
                    tq.id as task_query_id,
                    tq.query,
                    tq.task_id,
                    tj.platforms,
                    esql.sub_query,
                    esql.url,
                    esql.domain,
                    esql.title,
                    esql.snippet,
                    esql.site_name,
                    esql.cite_index,
                    esql.created_at
                FROM task_query tq
                INNER JOIN task_jobs tj ON tq.task_id = tj.id
                LEFT JOIN executor_sub_query_log esql ON tq.id = esql.task_query_id
                WHERE tq.task_id IN ({placeholders})
                ORDER BY tq.task_id, tq.id, esql.created_at
            """, task_ids)
            
            rows = cur.fetchall()
        
        # 生成CSV
        output = io.StringIO()
        writer = csv.writer(output)
        
        # 写入表头
        writer.writerow([
            '任务ID',
            '原始Query',
            '平台',
            'Sub Query',
            '网址',
            '域名',
            '标题',
            '摘要',
            '站点名称',
            '引用序号',
            '创建时间'
        ])
        
        # 写入数据
        for row in rows:
            task_query_id, query, task_id, platforms_json, sub_query, url, domain, title, snippet, site_name, cite_index, created_at = row
            
            # 解析平台列表
            if isinstance(platforms_json, (list, dict)):
                platforms = platforms_json
            else:
                platforms = json.loads(platforms_json) if platforms_json else []
            
            platforms_str = ', '.join(platforms) if isinstance(platforms, list) else str(platforms)
            
            # 修复编码
            query = ensure_utf8_string(query) if isinstance(query, str) else query
            sub_query = ensure_utf8_string(sub_query) if sub_query and isinstance(sub_query, str) else sub_query
            url = ensure_utf8_string(url) if url and isinstance(url, str) else url
            domain = ensure_utf8_string(domain) if domain and isinstance(domain, str) else domain
            title = ensure_utf8_string(title) if title and isinstance(title, str) else title
            snippet = ensure_utf8_string(snippet) if snippet and isinstance(snippet, str) else snippet
            site_name = ensure_utf8_string(site_name) if site_name and isinstance(site_name, str) else site_name
            
            writer.writerow([
                task_id,
                query or '',
                platforms_str,
                sub_query or '',
                url or '',
                domain or '',
                title or '',
                snippet or '',
                site_name or '',
                cite_index or '',
                created_at.isoformat() if created_at else ''
            ])
        
        # 准备响应
        csv_content = output.getvalue()
        output.close()
        
        # 生成文件名
        filename = f"task_data_{'_'.join(map(str, task_ids))}.csv"
        
        return Response(
            content=csv_content.encode('utf-8-sig'),  # 使用 utf-8-sig 以支持 Excel 正确显示中文
            media_type='text/csv',
            headers={
                'Content-Disposition': f'attachment; filename="{filename}"'
            }
        )
        
    except HTTPException:
        raise
    except Exception as e:
        logger.error(f"导出数据失败: {e}", exc_info=True)
        raise HTTPException(status_code=500, detail=f"导出数据失败: {str(e)}")


@app.post("/bocha/search")
async def bocha_search(query: str = Query(..., description="查询词")):
    """
    使用博查API进行实时搜索
    
    - **query**: 要搜索的查询词
    
    返回博查API的搜索结果，用于前端抽屉展示
    """
    try:
        if not query or not query.strip():
            raise HTTPException(status_code=400, detail="query 不能为空")
        
        # 创建博查Provider实例
        provider = BochaApiProvider(headless=True, timeout=30000)
        
        # 调用搜索
        result = provider.search(query, query)
        
        # 检查是否有错误
        if not result.get("citations") and not result.get("full_text"):
            return {
                "success": False,
                "error": "未获取到搜索结果",
                "data": result
            }
        
        return {
            "success": True,
            "data": {
                "full_text": result.get("full_text", ""),
                "queries": result.get("queries", []),
                "citations": result.get("citations", [])
            }
        }
        
    except HTTPException:
        raise
    except Exception as e:
        logger.error(f"博查搜索失败: {e}", exc_info=True)
        return {
            "success": False,
            "error": f"博查搜索失败: {str(e)}",
            "data": None
        }


if __name__ == "__main__":
    import uvicorn
    port = int(os.getenv("API_PORT", "8000"))
    uvicorn.run(app, host="0.0.0.0", port=port)

