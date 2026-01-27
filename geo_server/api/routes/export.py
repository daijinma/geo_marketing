"""
Data export API routes
"""
import logging
from fastapi import APIRouter, HTTPException, Query
from fastapi.responses import Response
from services import export_tasks_to_csv

logger = logging.getLogger(__name__)
router = APIRouter(tags=["export"])


@router.get("/export")
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
        
        # 生成CSV内容
        csv_content = export_tasks_to_csv(task_ids)
        
        # 生成文件名
        filename = f"task_data_{'_'.join(map(str, task_ids))}.csv"
        
        return Response(
            content=csv_content.encode('utf-8-sig'),  # 使用 utf-8-sig 以支持 Excel 正确显示中文
            media_type='text/csv',
            headers={
                'Content-Disposition': f'attachment; filename="{filename}"'
            }
        )
        
    except ValueError as e:
        raise HTTPException(status_code=400, detail=str(e))
    except Exception as e:
        logger.error(f"导出数据失败: {e}", exc_info=True)
        raise HTTPException(status_code=500, detail=f"导出数据失败: {str(e)}")
