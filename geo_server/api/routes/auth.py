"""
Authentication API routes
"""
import logging
from datetime import datetime, timedelta
from fastapi import APIRouter, HTTPException
from models import LoginRequest, LoginResponse
from core.auth import authenticate_user, create_access_token

logger = logging.getLogger(__name__)
router = APIRouter(prefix="/client/auth", tags=["auth"])


@router.post("/login", response_model=LoginResponse)
async def login(credentials: LoginRequest):
    """
    客户端登录接口
    
    - **username**: 用户名
    - **password**: 密码
    
    返回:
    - **success**: 登录是否成功
    - **token**: JWT访问token
    - **expires_at**: token过期时间（ISO格式字符串）
    """
    try:
        # 验证用户凭据
        user = authenticate_user(credentials.username, credentials.password)
        
        if not user:
            raise HTTPException(
                status_code=401,
                detail="用户名或密码错误"
            )
        
        # 创建JWT token
        token_data = {
            "sub": str(user["id"]),  # subject (用户ID)
            "username": user["username"],
            "is_admin": user.get("is_admin", False)
        }
        
        access_token = create_access_token(data=token_data)
        
        # 计算过期时间（24小时后）
        expires_at = datetime.utcnow() + timedelta(hours=24)
        expires_at_str = expires_at.isoformat()
        
        logger.info(f"用户 {credentials.username} 登录成功")
        
        return LoginResponse(
            success=True,
            token=access_token,
            expires_at=expires_at_str
        )
        
    except HTTPException:
        raise
    except Exception as e:
        logger.error(f"登录失败: {e}", exc_info=True)
        raise HTTPException(
            status_code=500,
            detail=f"登录失败: {str(e)}"
        )
