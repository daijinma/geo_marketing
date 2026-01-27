"""
Health check API routes
"""
import logging
from fastapi import APIRouter, Query, HTTPException
from core.db import get_db_connection
from providers.bocha_api import BochaApiProvider

logger = logging.getLogger(__name__)
router = APIRouter(tags=["health"])


@router.get("/health")
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


@router.post("/bocha/search")
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
            "data": result
        }
        
    except Exception as e:
        logger.error(f"博查搜索失败: {e}", exc_info=True)
        return {
            "success": False,
            "error": str(e)
        }
