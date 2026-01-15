-- ============================================
-- 数据库升级脚本：添加任务关联关系 v3.1
-- 建立 task_jobs 与 search_records 的关联
-- 建立 executor_sub_query_log 与 citations 的关联
-- ============================================

BEGIN;

-- ============================================
-- 0. 确保 task_query 表存在（如果从 v2.1 升级，可能不存在）
-- ============================================
CREATE TABLE IF NOT EXISTS task_query (
    id SERIAL PRIMARY KEY,
    task_id INTEGER REFERENCES task_jobs(id) ON DELETE CASCADE,
    query TEXT NOT NULL,                        -- 用户输入的原始查询条件
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- 创建 task_query 表的索引（如果不存在）
CREATE INDEX IF NOT EXISTS idx_task_query_task_id ON task_query(task_id);
CREATE INDEX IF NOT EXISTS idx_task_query_query ON task_query(query);
CREATE INDEX IF NOT EXISTS idx_task_query_created_at ON task_query(created_at DESC);

-- 确保 executor_sub_query_log 表存在（如果从 v2.1 升级，可能不存在）
CREATE TABLE IF NOT EXISTS executor_sub_query_log (
    id SERIAL PRIMARY KEY,
    task_query_id INTEGER REFERENCES task_query(id) ON DELETE CASCADE,
    sub_query TEXT,                            -- 分词后的子查询（sub_query），可能为0～多个，可为空
    url TEXT,                                   -- 网址，可为空（当只有 sub_query 时）
    domain TEXT,                                -- 域名
    title TEXT,                                 -- 网页标题
    snippet TEXT,                               -- 内容摘要
    site_name TEXT,                             -- 站点名称
    cite_index INTEGER,                         -- 引用序号
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- 添加约束（如果不存在）
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.table_constraints 
        WHERE constraint_name = 'check_sub_query_or_url' AND table_name = 'executor_sub_query_log'
    ) THEN
        ALTER TABLE executor_sub_query_log
        ADD CONSTRAINT check_sub_query_or_url CHECK (sub_query IS NOT NULL OR url IS NOT NULL);
    END IF;
END $$;

-- 创建 executor_sub_query_log 表的基础索引（如果不存在）
CREATE INDEX IF NOT EXISTS idx_executor_sub_query_log_task_query_id ON executor_sub_query_log(task_query_id);
CREATE INDEX IF NOT EXISTS idx_executor_sub_query_log_url ON executor_sub_query_log(url) WHERE url IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_executor_sub_query_log_domain ON executor_sub_query_log(domain) WHERE domain IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_executor_sub_query_log_created_at ON executor_sub_query_log(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_executor_sub_query_log_sub_query ON executor_sub_query_log(sub_query) WHERE sub_query IS NOT NULL;

-- ============================================
-- 1. 在 search_records 表中添加任务关联字段
-- ============================================
ALTER TABLE search_records 
ADD COLUMN IF NOT EXISTS task_id INTEGER REFERENCES task_jobs(id) ON DELETE SET NULL,
ADD COLUMN IF NOT EXISTS task_query_id INTEGER REFERENCES task_query(id) ON DELETE SET NULL;

-- ============================================
-- 2. 在 executor_sub_query_log 表中添加关联字段
-- ============================================
-- 添加 search_records 关联
ALTER TABLE executor_sub_query_log 
ADD COLUMN IF NOT EXISTS record_id INTEGER REFERENCES search_records(id) ON DELETE CASCADE;

-- 添加 citations 关联（避免重复存储）
ALTER TABLE executor_sub_query_log 
ADD COLUMN IF NOT EXISTS citation_id INTEGER REFERENCES citations(id) ON DELETE SET NULL;

-- ============================================
-- 3. 创建索引以提升查询性能
-- ============================================
-- search_records 表的任务关联索引
CREATE INDEX IF NOT EXISTS idx_search_records_task_id ON search_records(task_id) WHERE task_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_search_records_task_query_id ON search_records(task_query_id) WHERE task_query_id IS NOT NULL;

-- executor_sub_query_log 表的关联索引
CREATE INDEX IF NOT EXISTS idx_executor_sub_query_log_record_id ON executor_sub_query_log(record_id) WHERE record_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_executor_sub_query_log_citation_id ON executor_sub_query_log(citation_id) WHERE citation_id IS NOT NULL;

-- ============================================
-- 4. 版本记录
-- ============================================
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

INSERT INTO schema_version (version, description) 
VALUES ('3.1', '添加任务关联关系：search_records 关联 task_jobs/task_query，executor_sub_query_log 关联 search_records/citations')
ON CONFLICT (version) DO NOTHING;

COMMIT;

-- ============================================
-- 完成后验证
-- ============================================
SELECT 'Migration 003 completed successfully!' as status;
SELECT version, applied_at, description FROM schema_version ORDER BY applied_at DESC LIMIT 5;

