-- ============================================
-- 为 geo_sentry 用户创建和授权脚本
-- 修复 "permission denied for table" 错误
-- ============================================

-- 创建 geo_sentry 用户（如果不存在）
DO $$
BEGIN
    IF NOT EXISTS (SELECT FROM pg_catalog.pg_user WHERE usename = 'geo_sentry') THEN
        CREATE USER geo_sentry WITH PASSWORD 'geo_password123';
        RAISE NOTICE '用户 geo_sentry 已创建';
    ELSE
        RAISE NOTICE '用户 geo_sentry 已存在';
    END IF;
END $$;

-- 授予 geo_sentry 用户所有表的完整权限
GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA public TO geo_sentry;
GRANT ALL PRIVILEGES ON ALL SEQUENCES IN SCHEMA public TO geo_sentry;

-- 确保未来创建的表和序列也自动授予权限
ALTER DEFAULT PRIVILEGES IN SCHEMA public 
    GRANT ALL PRIVILEGES ON TABLES TO geo_sentry;
ALTER DEFAULT PRIVILEGES IN SCHEMA public 
    GRANT ALL PRIVILEGES ON SEQUENCES TO geo_sentry;

-- 授予函数执行权限（用于触发器函数）
GRANT EXECUTE ON FUNCTION update_updated_at_column() TO geo_sentry;

-- 验证权限
SELECT 'geo_sentry 用户权限已授予成功!' as status;

