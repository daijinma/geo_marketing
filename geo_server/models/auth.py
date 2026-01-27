"""
Authentication-related Pydantic models
"""
from pydantic import BaseModel


class LoginRequest(BaseModel):
    """用户登录请求"""
    username: str
    password: str


class LoginResponse(BaseModel):
    """用户登录响应"""
    success: bool
    token: str
    expires_at: str
