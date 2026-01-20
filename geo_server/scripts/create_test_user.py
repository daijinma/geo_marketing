#!/usr/bin/env python3
"""
创建测试用户脚本
用于初始化测试用户，方便联调测试
"""
import sys
import os

# 添加项目根目录到Python路径
sys.path.insert(0, os.path.dirname(os.path.dirname(os.path.abspath(__file__))))

from core.auth import create_user
from dotenv import load_dotenv

# 加载环境变量
env_file = os.getenv("ENV_FILE", ".env")
load_dotenv(env_file)


def main():
    """创建测试用户"""
    print("=" * 60)
    print("创建测试用户")
    print("=" * 60)
    
    # 默认测试用户信息
    username = input("请输入用户名（默认: admin）: ").strip() or "admin"
    password = input("请输入密码（默认: admin123）: ").strip() or "admin123"
    email = input("请输入邮箱（可选，按Enter跳过）: ").strip() or None
    full_name = input("请输入全名（可选，按Enter跳过）: ").strip() or None
    
    # 询问是否设置为管理员
    is_admin_input = input("是否设置为管理员？(y/N): ").strip().lower()
    is_admin = is_admin_input == 'y'
    
    try:
        user_id = create_user(
            username=username,
            password=password,
            email=email,
            full_name=full_name,
            is_admin=is_admin
        )
        
        if user_id:
            print(f"\n✅ 用户创建成功！")
            print(f"   用户ID: {user_id}")
            print(f"   用户名: {username}")
            print(f"   是否管理员: {'是' if is_admin else '否'}")
            print(f"\n可以使用以下凭据登录：")
            print(f"   用户名: {username}")
            print(f"   密码: {password}")
        else:
            print("\n❌ 用户创建失败，可能用户名已存在")
            sys.exit(1)
            
    except Exception as e:
        print(f"\n❌ 创建用户时发生错误: {e}")
        sys.exit(1)


if __name__ == "__main__":
    main()
