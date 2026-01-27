-- ============================================
-- 数据库升级脚本：添加任务创建人和类型字段 v3.0
-- ============================================

BEGIN;

-- 1. 添加创建人和任务类型字段到 task_jobs
ALTER TABLE task_jobs 
  ADD COLUMN IF NOT EXISTS user_id INTEGER,
  ADD COLUMN IF NOT EXISTS created_by VARCHAR(100),
  ADD COLUMN IF NOT EXISTS task_type VARCHAR(50) DEFAULT 'local_search',
  ADD COLUMN IF NOT EXISTS source VARCHAR(20) DEFAULT 'local';

-- 2. 添加外键约束（关联users表）
ALTER TABLE task_jobs
  ADD CONSTRAINT task_jobs_user_id_fkey 
    FOREIGN KEY (user_id) 
    REFERENCES users(id) 
    ON DELETE SET NULL;

-- 3. 创建索引以提升查询性能
CREATE INDEX IF NOT EXISTS idx_task_jobs_user_id ON task_jobs(user_id);
CREATE INDEX IF NOT EXISTS idx_task_jobs_created_by ON task_jobs(created_by);
CREATE INDEX IF NOT EXISTS idx_task_jobs_task_type ON task_jobs(task_type);
CREATE INDEX IF NOT EXISTS idx_task_jobs_source ON task_jobs(source);

-- 4. 添加注释
COMMENT ON COLUMN task_jobs.user_id IS '创建任务的用户ID（关联users表）';
COMMENT ON COLUMN task_jobs.created_by IS '创建任务的用户名';
COMMENT ON COLUMN task_jobs.task_type IS '任务类型：local_search, api_search, data_analysis, custom';
COMMENT ON COLUMN task_jobs.source IS '任务来源：local（客户端创建）, server（服务端下发）';

-- 5. 版本记录
INSERT INTO schema_version (version, description) 
VALUES ('3.0', '添加任务创建人和类型字段（user_id, created_by, task_type, source）')
ON CONFLICT (version) DO NOTHING;

COMMIT;

-- 完成后验证
SELECT 'Migration 007 completed successfully!' as status;
SELECT version, applied_at, description FROM schema_version ORDER BY applied_at DESC LIMIT 5;
