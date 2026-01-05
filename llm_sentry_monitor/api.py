"""
api.py - FastAPI 应用
提供 REST API 接口用于任务管理和状态查询
"""
import os
import json
import logging
import jieba
import jieba.analyse
from typing import List, Dict, Any, Optional
from fastapi import FastAPI, HTTPException, Query
from pydantic import BaseModel
from core.db import get_db_connection
from core.task_executor import execute_task_job

# 初始化 jieba 自定义词典（与 stats.py 保持一致）
CUSTOM_WORDS = [
    "土巴兔", "装修公司", "家装", "软装", "硬装", "全包", "半包",
    "DeepSeek", "豆包", "Kimi", "文心一言", "通义千问"
]

for word in CUSTOM_WORDS:
    jieba.add_word(word)

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
    - **settings**: 可选设置 (headless, timeout, delay_between_tasks)
    """
    try:
        # 验证输入
        if not request.keywords:
            raise HTTPException(status_code=400, detail="keywords 不能为空")
        
        if not request.platforms:
            raise HTTPException(status_code=400, detail="platforms 不能为空")
        
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
                INSERT INTO task_jobs (keywords, platforms, status, settings)
                VALUES (%s, %s, 'pending', %s)
                RETURNING id
            """, (
                json.dumps(request.keywords),
                json.dumps(request.platforms),
                json.dumps(settings)
            ))
            task_id = cur.fetchone()[0]
            conn.commit()
        
        logger.info(f"创建任务 {task_id}: keywords={request.keywords}, platforms={request.platforms}")
        
        # 启动后台任务
        execute_task_job(task_id, request.keywords, request.platforms, settings)
        
        return MockResponse(task_id=task_id)
        
    except HTTPException:
        raise
    except Exception as e:
        logger.error(f"创建任务失败: {e}", exc_info=True)
        raise HTTPException(status_code=500, detail=f"创建任务失败: {str(e)}")


@app.get("/status", response_model=StatusResponse)
async def get_task_status(id: int = Query(..., description="任务ID")):
    """
    查询任务状态
    
    - **id**: 任务ID
    
    返回:
    - **status**: 任务状态 (none, pending, done)
    - **data**: 任务数据（当 status != none 时）
    """
    try:
        with get_db_connection() as conn:
            cur = conn.cursor()
            cur.execute("""
                SELECT id, keywords, platforms, status, result_data, created_at, updated_at
                FROM task_jobs
                WHERE id = %s
            """, (id,))
            
            row = cur.fetchone()
            
            if not row:
                return StatusResponse(status="none", data=None)
            
            task_id, keywords_json, platforms_json, status, result_data_json, created_at, updated_at = row
            
            # 解析 JSON 数据
            # PostgreSQL JSONB 列会自动反序列化为 Python 对象，需要检查类型
            if isinstance(keywords_json, (list, dict)):
                keywords = keywords_json
            else:
                keywords = json.loads(keywords_json) if keywords_json else []
            
            if isinstance(platforms_json, (list, dict)):
                platforms = platforms_json
            else:
                platforms = json.loads(platforms_json) if platforms_json else []
            
            if isinstance(result_data_json, (list, dict)):
                result_data = result_data_json
            else:
                result_data = json.loads(result_data_json) if result_data_json else {}
            
            # 构建响应数据
            response_data = {
                "task_id": task_id,
                "keywords": keywords,
                "platforms": platforms,
                "created_at": created_at.isoformat() if created_at else None,
                "updated_at": updated_at.isoformat() if updated_at else None,
            }
            
            # 如果任务完成，添加结果
            if status == "done" and result_data:
                response_data["results"] = result_data
            
            # 查询内部查询的分词
            # 根据任务的 keywords 和 platforms 查找对应的 search_records
            query_tokens = []
            if keywords and platforms:
                # 构建查询条件：查找匹配的 search_records
                placeholders = []
                params = []
                
                # 为每个 keyword 和 platform 组合创建查询条件
                for keyword in keywords:
                    for platform in platforms:
                        placeholders.append("(keyword = %s AND platform = %s AND prompt_type = 'api_task')")
                        params.extend([keyword, platform.lower()])
                
                if placeholders:
                    # 查询所有相关的 search_queries
                    query_sql = f"""
                        SELECT DISTINCT sq.query
                        FROM search_queries sq
                        INNER JOIN search_records sr ON sq.record_id = sr.id
                        WHERE {' OR '.join(placeholders)}
                        ORDER BY sq.query_order, sq.id
                    """
                    cur.execute(query_sql, params)
                    queries = [row[0] for row in cur.fetchall() if row[0]]
                    
                    # 对每个查询进行分词
                    for query in queries:
                        if query:
                            # 使用 jieba 进行分词
                            tokens = jieba.lcut(query)
                            # 过滤掉单字符和空白
                            tokens = [t for t in tokens if len(t.strip()) > 1]
                            if tokens:
                                query_tokens.append({
                                    "query": query,
                                    "tokens": tokens
                                })
            
            # 添加分词结果到响应
            if query_tokens:
                response_data["query_tokens"] = query_tokens
            
            return StatusResponse(status=status, data=response_data)
            
    except Exception as e:
        logger.error(f"查询任务状态失败: {e}", exc_info=True)
        raise HTTPException(status_code=500, detail=f"查询任务状态失败: {str(e)}")


@app.get("/")
async def root():
    """API 根路径"""
    return {
        "message": "LLM Sentry Monitor API",
        "version": "1.0.0",
        "endpoints": {
            "POST /mock": "创建新的搜索任务",
            "GET /status?id=<task_id>": "查询任务状态"
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


if __name__ == "__main__":
    import uvicorn
    port = int(os.getenv("API_PORT", "8000"))
    uvicorn.run(app, host="0.0.0.0", port=port)

