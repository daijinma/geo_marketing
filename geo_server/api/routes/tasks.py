"""
Task-related API routes
"""
import os
import logging
from typing import Optional
from fastapi import APIRouter, HTTPException, Query, Header
from fastapi.responses import FileResponse
from models import MockRequest, MockResponse, StatusResponse, TaskSyncRequest, TaskSyncResponse
from services import create_task_job, get_task_status_data
from core.auth import decode_access_token
from core.db import get_db_connection
from datetime import datetime

logger = logging.getLogger(__name__)
router = APIRouter()


@router.get("/")
async def root():
    """前端页面入口"""
    static_dir_path = os.path.join(os.path.dirname(os.path.dirname(os.path.dirname(__file__))), "static")
    index_path = os.path.join(static_dir_path, "index.html")
    if os.path.exists(index_path):
        return FileResponse(index_path)
    # 如果没有前端页面，返回API信息
    return {
        "message": "LLM Sentry Monitor API",
        "version": "1.0.0",
        "endpoints": {
            "POST /client/auth/login": "客户端登录",
            "POST /mock": "创建新的搜索任务",
            "GET /status?id=<task_id>": "查询任务状态",
            "POST /bocha/search?query=<query>": "博查实时搜索"
        }
    }


@router.post("/mock", response_model=MockResponse)
async def create_task(request: MockRequest):
    """
    创建新的搜索任务
    
    - **keywords**: 搜索关键词列表
    - **platforms**: 平台列表 (deepseek, doubao)
    - **query_count**: 查询次数（执行轮数），默认1次
    - **settings**: 可选设置 (headless, timeout, delay_between_tasks)
    """
    try:
        task_id = create_task_job(
            keywords=request.keywords,
            platforms=request.platforms,
            query_count=request.query_count,
            settings=request.settings
        )
        return MockResponse(task_id=task_id)
        
    except ValueError as e:
        raise HTTPException(status_code=400, detail=str(e))
    except Exception as e:
        logger.error(f"创建任务失败: {e}", exc_info=True)
        raise HTTPException(status_code=500, detail=f"创建任务失败: {str(e)}")


@router.get("/status", response_model=StatusResponse)
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
        
        result = get_task_status_data(task_ids)
        return StatusResponse(status=result["status"], data=result.get("data"))
        
    except ValueError as e:
        raise HTTPException(status_code=400, detail=str(e))
    except Exception as e:
        logger.error(f"查询任务状态失败: {e}", exc_info=True)
        raise HTTPException(status_code=500, detail=f"查询任务状态失败: {str(e)}")


@router.post("/client/mytask", response_model=TaskSyncResponse)
async def sync_task_status(
    request: TaskSyncRequest,
    authorization: str = Header(None)
):
    """
    同步客户端任务状态到服务端
    
    - **tasks**: 任务列表
    
    需要认证（Bearer Token）
    """
    try:
        # 1. 验证token
        if not authorization or not authorization.startswith("Bearer "):
            raise HTTPException(status_code=401, detail="未提供认证token")
        
        token = authorization.split(" ")[1]
        user_data = decode_access_token(token)
        
        if not user_data:
            raise HTTPException(status_code=401, detail="Token无效或已过期")
        
        user_id = int(user_data["sub"])
        username = user_data["username"]
        
        logger.info(f"用户 {username} (ID: {user_id}) 请求同步 {len(request.tasks)} 个任务")
        
        # 2. 处理任务同步
        synced = []
        errors = []
        
        with get_db_connection() as conn:
            cur = conn.cursor()
            
            for task in request.tasks:
                try:
                    # 检查是否已存在该任务（通过 local_task_id 和 user_id）
                    cur.execute("""
                        SELECT id FROM task_jobs 
                        WHERE created_by = %s AND task_type = %s 
                        AND keywords = %s::jsonb
                        LIMIT 1
                    """, (username, task.task_type, f'["{task.keywords}"]'))
                    
                    existing = cur.fetchone()
                    
                    if existing:
                        # 已存在，更新状态
                        server_task_id = existing[0]
                        cur.execute("""
                            UPDATE task_jobs 
                            SET status = %s, updated_at = %s
                            WHERE id = %s
                        """, (task.status, datetime.utcnow(), server_task_id))
                        
                        synced.append({
                            "local_task_id": task.local_task_id,
                            "server_task_id": server_task_id
                        })
                        logger.info(f"更新任务: local_id={task.local_task_id}, server_id={server_task_id}")
                    else:
                        # 不存在，创建新任务
                        cur.execute("""
                            INSERT INTO task_jobs 
                            (keywords, platforms, query_count, status, created_by, task_type, created_at, updated_at)
                            VALUES (%s, %s, %s, %s, %s, %s, %s, %s)
                            RETURNING id
                        """, (
                            f'["{task.keywords}"]',  # JSON格式
                            f'["{task.platforms}"]',  # JSON格式
                            task.query_count,
                            task.status,
                            username,
                            task.task_type,
                            datetime.fromisoformat(task.created_at.replace('Z', '+00:00')) if task.created_at else datetime.utcnow(),
                            datetime.utcnow()
                        ))
                        
                        server_task_id = cur.fetchone()[0]
                        
                        # 创建 task_query 记录
                        cur.execute("""
                            INSERT INTO task_query (task_id, query)
                            VALUES (%s, %s)
                        """, (server_task_id, task.keywords))
                        
                        synced.append({
                            "local_task_id": task.local_task_id,
                            "server_task_id": server_task_id
                        })
                        logger.info(f"创建新任务: local_id={task.local_task_id}, server_id={server_task_id}")
                
                except Exception as e:
                    error_msg = f"同步任务 {task.local_task_id} 失败: {str(e)}"
                    logger.error(error_msg, exc_info=True)
                    errors.append(error_msg)
            
            conn.commit()
        
        return TaskSyncResponse(
            success=len(errors) == 0,
            synced=synced,
            errors=errors if errors else None
        )
        
    except HTTPException:
        raise
    except Exception as e:
        logger.error(f"任务同步失败: {e}", exc_info=True)
        raise HTTPException(status_code=500, detail=f"任务同步失败: {str(e)}")
