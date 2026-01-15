-- ============================================
-- 数据库升级脚本：添加任务管理表 v2.1
-- ============================================

BEGIN;

-- 创建任务表
CREATE TABLE IF NOT EXISTS task_jobs (
    id SERIAL PRIMARY KEY,
    keywords JSONB NOT NULL,                    -- 关键词数组
    platforms JSONB NOT NULL,                   -- 平台数组
    status TEXT DEFAULT 'pending',              -- 任务状态: none, pending, done
    result_data JSONB DEFAULT '{}',             -- 抓取结果数据
    settings JSONB DEFAULT '{}',                 -- 任务设置（headless, timeout等）
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- 创建索引
CREATE INDEX IF NOT EXISTS idx_task_jobs_status ON task_jobs(status);
CREATE INDEX IF NOT EXISTS idx_task_jobs_created_at ON task_jobs(created_at DESC);

-- 创建触发器：自动更新 updated_at（如果不存在）
DROP TRIGGER IF EXISTS update_task_jobs_updated_at ON task_jobs;
CREATE TRIGGER update_task_jobs_updated_at BEFORE UPDATE ON task_jobs
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- 版本记录
INSERT INTO schema_version (version, description) 
VALUES ('2.1', '添加任务管理表 task_jobs')
ON CONFLICT (version) DO NOTHING;

COMMIT;

-- 完成后验证
SELECT 'Migration 002 completed successfully!' as status;
SELECT version, applied_at, description FROM schema_version ORDER BY applied_at DESC LIMIT 5;


