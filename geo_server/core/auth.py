"""
core/auth.py - 用户认证工具模块
提供密码加密、验证和JWT token生成功能
"""
import os
import jwt
import bcrypt
from datetime import datetime, timedelta
from typing import Optional, Dict, Any
from core.db import get_db_connection

# JWT配置
JWT_SECRET_KEY = os.getenv("JWT_SECRET_KEY", "your-secret-key-change-in-production")
JWT_ALGORITHM = "HS256"
JWT_ACCESS_TOKEN_EXPIRE_HOURS = 24  # Token有效期24小时


def hash_password(password: str) -> str:
    """
    使用bcrypt加密密码
    
    Args:
        password: 原始密码字符串
        
    Returns:
        加密后的密码哈希值（字符串）
    """
    password_bytes = password.encode('utf-8')
    salt = bcrypt.gensalt()
    password_hash = bcrypt.hashpw(password_bytes, salt)
    return password_hash.decode('utf-8')


def verify_password(password: str, password_hash: str) -> bool:
    """
    验证密码是否匹配
    
    Args:
        password: 原始密码字符串
        password_hash: 存储的密码哈希值
        
    Returns:
        密码是否匹配
    """
    password_bytes = password.encode('utf-8')
    password_hash_bytes = password_hash.encode('utf-8')
    return bcrypt.checkpw(password_bytes, password_hash_bytes)


def create_access_token(data: Dict[str, Any], expires_delta: Optional[timedelta] = None) -> str:
    """
    创建JWT访问token
    
    Args:
        data: 要编码到token中的数据（如user_id, username）
        expires_delta: token过期时间差，如果不提供则使用默认值
        
    Returns:
        JWT token字符串
    """
    to_encode = data.copy()
    
    if expires_delta:
        expire = datetime.utcnow() + expires_delta
    else:
        expire = datetime.utcnow() + timedelta(hours=JWT_ACCESS_TOKEN_EXPIRE_HOURS)
    
    to_encode.update({"exp": expire, "iat": datetime.utcnow()})
    
    encoded_jwt = jwt.encode(to_encode, JWT_SECRET_KEY, algorithm=JWT_ALGORITHM)
    return encoded_jwt


def decode_access_token(token: str) -> Optional[Dict[str, Any]]:
    """
    解码JWT token
    
    Args:
        token: JWT token字符串
        
    Returns:
        解码后的token数据，如果token无效则返回None
    """
    try:
        payload = jwt.decode(token, JWT_SECRET_KEY, algorithms=[JWT_ALGORITHM])
        return payload
    except jwt.ExpiredSignatureError:
        return None
    except jwt.InvalidTokenError:
        return None


def authenticate_user(username: str, password: str) -> Optional[Dict[str, Any]]:
    """
    验证用户凭据
    
    Args:
        username: 用户名
        password: 密码
        
    Returns:
        如果验证成功，返回用户信息字典（包含id, username等），否则返回None
    """
    try:
        with get_db_connection() as conn:
            cur = conn.cursor()
            cur.execute("""
                SELECT id, username, password_hash, email, full_name, is_active, is_admin
                FROM users
                WHERE username = %s
            """, (username,))
            
            row = cur.fetchone()
            
            if not row:
                return None
            
            user_id, db_username, password_hash, email, full_name, is_active, is_admin = row
            
            # 检查用户是否激活
            if not is_active:
                return None
            
            # 验证密码
            if not verify_password(password, password_hash):
                return None
            
            # 更新最后登录时间
            cur.execute("""
                UPDATE users
                SET last_login_at = CURRENT_TIMESTAMP
                WHERE id = %s
            """, (user_id,))
            
            return {
                "id": user_id,
                "username": db_username,
                "email": email,
                "full_name": full_name,
                "is_admin": is_admin
            }
    except Exception as e:
        print(f"认证用户失败: {e}")
        return None


def get_user_by_id(user_id: int) -> Optional[Dict[str, Any]]:
    """
    根据用户ID获取用户信息
    
    Args:
        user_id: 用户ID
        
    Returns:
        用户信息字典，如果用户不存在则返回None
    """
    try:
        with get_db_connection() as conn:
            cur = conn.cursor()
            cur.execute("""
                SELECT id, username, email, full_name, is_active, is_admin
                FROM users
                WHERE id = %s
            """, (user_id,))
            
            row = cur.fetchone()
            
            if not row:
                return None
            
            user_id, username, email, full_name, is_active, is_admin = row
            
            return {
                "id": user_id,
                "username": username,
                "email": email,
                "full_name": full_name,
                "is_active": is_active,
                "is_admin": is_admin
            }
    except Exception as e:
        print(f"获取用户信息失败: {e}")
        return None


def create_user(username: str, password: str, email: Optional[str] = None, 
                full_name: Optional[str] = None, is_admin: bool = False) -> Optional[int]:
    """
    创建新用户（用于初始化或管理）
    
    Args:
        username: 用户名
        password: 密码
        email: 邮箱（可选）
        full_name: 全名（可选）
        is_admin: 是否管理员
        
    Returns:
        新创建的用户ID，如果创建失败则返回None
    """
    try:
        password_hash = hash_password(password)
        
        with get_db_connection() as conn:
            cur = conn.cursor()
            cur.execute("""
                INSERT INTO users (username, password_hash, email, full_name, is_admin)
                VALUES (%s, %s, %s, %s, %s)
                RETURNING id
            """, (username, password_hash, email, full_name, is_admin))
            
            user_id = cur.fetchone()[0]
            return user_id
    except Exception as e:
        print(f"创建用户失败: {e}")
        return None
