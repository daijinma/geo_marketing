"""
Task-related Pydantic models for request/response validation
"""
from typing import List, Dict, Any, Optional
from pydantic import BaseModel


class MockRequest(BaseModel):
    """创建搜索任务的请求模型"""
    keywords: List[str]
    platforms: List[str] = ["deepseek"]
    query_count: int = 1  # 查询次数（执行轮数），默认1次
    settings: Optional[Dict[str, Any]] = None


class MockResponse(BaseModel):
    """创建任务的响应模型"""
    task_id: int


class StatusResponse(BaseModel):
    """任务状态查询的响应模型"""
    status: str  # none, pending, done
    data: Optional[Dict[str, Any]] = None


class TaskSyncItem(BaseModel):
    """单个任务同步项"""
    local_task_id: int
    task_type: str
    keywords: str
    platforms: str
    query_count: int
    status: str
    created_at: str
    source: str = "local"
    created_by: Optional[str] = None


class TaskSyncRequest(BaseModel):
    """批量任务同步请求"""
    tasks: List[TaskSyncItem]


class TaskSyncResponse(BaseModel):
    """批量任务同步响应"""
    success: bool
    synced: List[Dict[str, int]]
    errors: Optional[List[str]] = None
