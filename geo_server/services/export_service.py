"""
Export service - Business logic for data export
"""
import json
import logging
import csv
import io
from typing import List
from core.db import get_db_connection
from utils.encoding import ensure_utf8_string

logger = logging.getLogger(__name__)


def export_tasks_to_csv(task_ids: List[int]) -> str:
    """
    导出任务明细数据为CSV格式
    
    Args:
        task_ids: 任务ID列表
    
    Returns:
        CSV内容字符串（UTF-8-SIG编码）
    
    Raises:
        ValueError: 参数验证失败
        Exception: 数据库查询失败
    """
    if not task_ids:
        raise ValueError("任务ID列表不能为空")
    
    # 查询数据
    with get_db_connection() as conn:
        cur = conn.cursor()
        
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
    
    csv_content = output.getvalue()
    output.close()
    
    return csv_content
