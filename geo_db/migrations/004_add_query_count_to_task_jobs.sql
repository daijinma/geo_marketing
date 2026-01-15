-- ============================================
-- 数据库升级脚本：为 task_jobs 表添加 query_count 字段 v2.2
-- ============================================

BEGIN;

-- 添加 query_count 字段（如果不存在）
ALTER TABLE task_jobs 
ADD COLUMN IF NOT EXISTS query_count INTEGER DEFAULT 1;

-- 创建索引（如果不存在）
CREATE INDEX IF NOT EXISTS idx_task_jobs_query_count ON task_jobs(query_count);

-- 确保 schema_version 表有 UNIQUE 约束（如果不存在）
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.table_constraints 
        WHERE constraint_name = 'schema_version_version_key' AND table_name = 'schema_version'
    ) THEN
        ALTER TABLE schema_version 
        ADD CONSTRAINT schema_version_version_key UNIQUE (version);
    END IF;
END $$;

-- 版本记录
INSERT INTO schema_version (version, description) 
VALUES ('2.2', '为 task_jobs 表添加 query_count 字段')
ON CONFLICT (version) DO NOTHING;

COMMIT;

-- ============================================
-- 完成后验证
-- ============================================
SELECT 'Migration 004 completed successfully!' as status;
SELECT version, applied_at, description FROM schema_version ORDER BY applied_at DESC LIMIT 5;

