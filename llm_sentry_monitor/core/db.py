"""
core/db.py - 统一数据库配置和连接管理
"""
import os
import psycopg2
from psycopg2.extras import RealDictCursor
from contextlib import contextmanager
from dotenv import load_dotenv

load_dotenv()

DB_CONFIG = {
    "host": os.getenv("DB_HOST", "localhost"),
    "port": os.getenv("DB_PORT", "5432"),
    "database": os.getenv("DB_NAME", "geo_monitor"),
    "user": os.getenv("DB_USER", "geo_admin"),
    "password": os.getenv("DB_PASSWORD", "geo_password123")
}

@contextmanager
def get_db_connection():
    """上下文管理器：自动管理数据库连接"""
    conn = None
    try:
        conn = psycopg2.connect(**DB_CONFIG)
        # 设置客户端编码为 UTF-8，确保正确处理中文字符
        conn.set_client_encoding('UTF8')
        yield conn
        conn.commit()
    except Exception as e:
        if conn:
            conn.rollback()
        raise e
    finally:
        if conn:
            conn.close()

def get_db_cursor(dict_cursor=False):
    """获取数据库游标"""
    conn = psycopg2.connect(**DB_CONFIG)
    # 设置客户端编码为 UTF-8，确保正确处理中文字符
    conn.set_client_encoding('UTF8')
    if dict_cursor:
        return conn, conn.cursor(cursor_factory=RealDictCursor)
    return conn, conn.cursor()

def update_domain_stats(conn, domain, platform):
    """更新域名统计信息"""
    cur = conn.cursor()
    cur.execute("""
        INSERT INTO domain_stats (domain, total_citations, keyword_coverage, platforms, last_seen)
        VALUES (%s, 1, 1, %s::jsonb, CURRENT_TIMESTAMP)
        ON CONFLICT (domain) DO UPDATE SET
            total_citations = domain_stats.total_citations + 1,
            platforms = domain_stats.platforms || %s::jsonb,
            last_seen = CURRENT_TIMESTAMP
    """, (domain, f'{{"{platform}": 1}}', f'{{"{platform}": 1}}'))
    cur.close()
