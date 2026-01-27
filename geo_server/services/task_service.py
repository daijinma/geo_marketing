"""
Task service - Business logic for task creation and management
"""
import json
import logging
from typing import List, Dict, Any, Optional
from core.db import get_db_connection
from core.task_executor import execute_task_job

logger = logging.getLogger(__name__)


def create_task_job(
    keywords: List[str],
    platforms: List[str],
    query_count: int,
    settings: Optional[Dict[str, Any]] = None
) -> int:
    """
    创建新的搜索任务
    
    Args:
        keywords: 搜索关键词列表
        platforms: 平台列表
        query_count: 查询次数（执行轮数）
        settings: 可选设置（headless, timeout, delay_between_tasks等）
    
    Returns:
        task_id: 新创建的任务ID
    
    Raises:
        ValueError: 参数验证失败
        Exception: 数据库操作失败或任务执行失败
    """
    # 验证输入
    if not keywords:
        raise ValueError("keywords 不能为空")
    
    if not platforms:
        raise ValueError("platforms 不能为空")
    
    if query_count < 1:
        raise ValueError("query_count 必须大于0")
    
    # 加载默认设置
    default_settings = {
        "headless": False,
        "timeout": 60000,
        "delay_between_tasks": 5
    }
    
    # 合并用户设置
    merged_settings = {**default_settings, **(settings or {})}
    
    # 创建任务记录
    with get_db_connection() as conn:
        cur = conn.cursor()
        cur.execute("""
            INSERT INTO task_jobs (keywords, platforms, query_count, status, settings)
            VALUES (%s, %s, %s, 'pending', %s)
            RETURNING id
        """, (
            json.dumps(keywords),
            json.dumps(platforms),
            query_count,
            json.dumps(merged_settings)
        ))
        task_id = cur.fetchone()[0]
        
        # 为每个查询条件创建 task_query 记录
        for keyword in keywords:
            cur.execute("""
                INSERT INTO task_query (task_id, query)
                VALUES (%s, %s)
            """, (task_id, keyword))
        
        conn.commit()
    
    logger.info(f"创建任务 {task_id}: keywords={keywords}, platforms={platforms}, query_count={query_count}")
    
    # 启动后台任务
    execute_task_job(task_id, keywords, platforms, query_count, merged_settings)
    
    return task_id
