-- ============================================
-- 数据库完整升级脚本：从 v1.0 升级到 v3.1
-- 整合所有迁移步骤，一次性完成所有更新
-- ============================================

BEGIN;

-- ============================================
-- 第一部分：v2.0 升级 - 优化现有表结构
-- ============================================

-- 1. 增强主记录表
ALTER TABLE search_records 
ADD COLUMN IF NOT EXISTS prompt TEXT,
ADD COLUMN IF NOT EXISTS response_time_ms INTEGER,
ADD COLUMN IF NOT EXISTS search_status TEXT DEFAULT 'completed',
ADD COLUMN IF NOT EXISTS error_message TEXT,
ADD COLUMN IF NOT EXISTS updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP;

-- 2. 增强拓展词表
ALTER TABLE search_queries 
ADD COLUMN IF NOT EXISTS query_order INTEGER;

-- 修改外键为级联删除
DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM information_schema.table_constraints 
        WHERE constraint_name = 'search_queries_record_id_fkey'
    ) THEN
        ALTER TABLE search_queries 
        DROP CONSTRAINT search_queries_record_id_fkey;
    END IF;
END $$;

ALTER TABLE search_queries 
ADD CONSTRAINT search_queries_record_id_fkey 
    FOREIGN KEY (record_id) 
    REFERENCES search_records(id) 
    ON DELETE CASCADE;

-- 3. 增强引用表
DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM information_schema.table_constraints 
        WHERE constraint_name = 'citations_record_id_fkey'
    ) THEN
        ALTER TABLE citations 
        DROP CONSTRAINT citations_record_id_fkey;
    END IF;
END $$;

ALTER TABLE citations 
ADD CONSTRAINT citations_record_id_fkey 
    FOREIGN KEY (record_id) 
    REFERENCES search_records(id) 
    ON DELETE CASCADE;

-- 添加唯一约束（如果已有重复数据，需先清理）
-- 先删除重复的 citations（保留 id 最小的）
DELETE FROM citations 
WHERE id IN (
    SELECT id FROM (
        SELECT id, ROW_NUMBER() OVER (PARTITION BY record_id, url ORDER BY id) AS rn
        FROM citations
    ) t WHERE rn > 1
);

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.table_constraints 
        WHERE constraint_name = 'unique_citation'
    ) THEN
        ALTER TABLE citations 
        ADD CONSTRAINT unique_citation UNIQUE (record_id, url);
    END IF;
END $$;

-- 4. 创建域名统计表
CREATE TABLE IF NOT EXISTS domain_stats (
    id SERIAL PRIMARY KEY,
    domain TEXT UNIQUE NOT NULL,
    total_citations INTEGER DEFAULT 0,
    keyword_coverage INTEGER DEFAULT 0,
    platforms JSONB DEFAULT '{}',
    first_seen TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    last_seen TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- 5. 初始化域名统计数据
INSERT INTO domain_stats (domain, total_citations, keyword_coverage, first_seen, last_seen)
SELECT 
    domain,
    COUNT(*) as total_citations,
    COUNT(DISTINCT record_id) as keyword_coverage,
    MIN(created_at) as first_seen,
    MAX(created_at) as last_seen
FROM citations
GROUP BY domain
ON CONFLICT (domain) DO NOTHING;

-- 6. 创建触发器函数（如果不存在）
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- 7. 创建触发器
DROP TRIGGER IF EXISTS update_search_records_updated_at ON search_records;
CREATE TRIGGER update_search_records_updated_at BEFORE UPDATE ON search_records
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

DROP TRIGGER IF EXISTS update_domain_stats_updated_at ON domain_stats;
CREATE TRIGGER update_domain_stats_updated_at BEFORE UPDATE ON domain_stats
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- 8. 创建版本管理表（如果不存在）
CREATE TABLE IF NOT EXISTS schema_version (
    id SERIAL PRIMARY KEY,
    version TEXT UNIQUE NOT NULL,
    applied_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    description TEXT
);

-- ============================================
-- 第二部分：v2.1 升级 - 添加任务管理表
-- ============================================

-- 创建任务管理表
CREATE TABLE IF NOT EXISTS task_jobs (
    id SERIAL PRIMARY KEY,
    keywords JSONB NOT NULL,                    -- 关键词数组
    platforms JSONB NOT NULL,                   -- 平台数组
    status TEXT DEFAULT 'pending',              -- 任务状态: none, pending, done
    result_data JSONB DEFAULT '{}',             -- 抓取结果数据
    settings JSONB DEFAULT '{}',                -- 任务设置（headless, timeout等）
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- 创建任务查询关联表
CREATE TABLE IF NOT EXISTS task_query (
    id SERIAL PRIMARY KEY,
    task_id INTEGER REFERENCES task_jobs(id) ON DELETE CASCADE,
    query TEXT NOT NULL,                        -- 用户输入的原始查询条件
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- 创建执行器子查询日志表
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
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    -- 约束：至少要有 sub_query 或 url 之一
    CONSTRAINT check_sub_query_or_url CHECK (sub_query IS NOT NULL OR url IS NOT NULL)
);

-- 创建触发器：自动更新 task_jobs 的 updated_at
DROP TRIGGER IF EXISTS update_task_jobs_updated_at ON task_jobs;
CREATE TRIGGER update_task_jobs_updated_at BEFORE UPDATE ON task_jobs
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- ============================================
-- 第三部分：v2.2 升级 - 添加 query_count 字段
-- ============================================

-- 添加 query_count 字段（如果不存在）
ALTER TABLE task_jobs 
ADD COLUMN IF NOT EXISTS query_count INTEGER DEFAULT 1;

-- ============================================
-- 第四部分：v3.1 升级 - 添加任务关联关系
-- ============================================

-- 1. 在 search_records 表中添加任务关联字段
ALTER TABLE search_records 
ADD COLUMN IF NOT EXISTS task_id INTEGER REFERENCES task_jobs(id) ON DELETE SET NULL,
ADD COLUMN IF NOT EXISTS task_query_id INTEGER REFERENCES task_query(id) ON DELETE SET NULL;

-- 2. 在 executor_sub_query_log 表中添加关联字段
ALTER TABLE executor_sub_query_log 
ADD COLUMN IF NOT EXISTS record_id INTEGER REFERENCES search_records(id) ON DELETE CASCADE,
ADD COLUMN IF NOT EXISTS citation_id INTEGER REFERENCES citations(id) ON DELETE SET NULL;

-- ============================================
-- 第五部分：创建所有索引
-- ============================================

-- search_records 表索引
CREATE INDEX IF NOT EXISTS idx_search_records_keyword ON search_records(keyword);
CREATE INDEX IF NOT EXISTS idx_search_records_platform ON search_records(platform);
CREATE INDEX IF NOT EXISTS idx_search_records_created_at ON search_records(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_search_records_status ON search_records(search_status);
CREATE INDEX IF NOT EXISTS idx_search_records_task_id ON search_records(task_id) WHERE task_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_search_records_task_query_id ON search_records(task_query_id) WHERE task_query_id IS NOT NULL;

-- search_queries 表索引
CREATE INDEX IF NOT EXISTS idx_search_queries_record ON search_queries(record_id);
CREATE INDEX IF NOT EXISTS idx_search_queries_query ON search_queries(query);

-- citations 表索引
CREATE INDEX IF NOT EXISTS idx_citations_record ON citations(record_id);
CREATE INDEX IF NOT EXISTS idx_citations_domain ON citations(domain);
CREATE INDEX IF NOT EXISTS idx_citations_domain_created ON citations(domain, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_citations_url ON citations(url);

-- domain_stats 表索引
CREATE INDEX IF NOT EXISTS idx_domain_stats_domain ON domain_stats(domain);
CREATE INDEX IF NOT EXISTS idx_domain_stats_total ON domain_stats(total_citations DESC);
CREATE INDEX IF NOT EXISTS idx_domain_stats_coverage ON domain_stats(keyword_coverage DESC);

-- task_jobs 表索引
CREATE INDEX IF NOT EXISTS idx_task_jobs_status ON task_jobs(status);
CREATE INDEX IF NOT EXISTS idx_task_jobs_created_at ON task_jobs(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_task_jobs_query_count ON task_jobs(query_count);

-- task_query 表索引
CREATE INDEX IF NOT EXISTS idx_task_query_task_id ON task_query(task_id);
CREATE INDEX IF NOT EXISTS idx_task_query_query ON task_query(query);
CREATE INDEX IF NOT EXISTS idx_task_query_created_at ON task_query(created_at DESC);

-- executor_sub_query_log 表索引
CREATE INDEX IF NOT EXISTS idx_executor_sub_query_log_task_query_id ON executor_sub_query_log(task_query_id);
CREATE INDEX IF NOT EXISTS idx_executor_sub_query_log_url ON executor_sub_query_log(url) WHERE url IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_executor_sub_query_log_domain ON executor_sub_query_log(domain) WHERE domain IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_executor_sub_query_log_created_at ON executor_sub_query_log(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_executor_sub_query_log_sub_query ON executor_sub_query_log(sub_query) WHERE sub_query IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_executor_sub_query_log_record_id ON executor_sub_query_log(record_id) WHERE record_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_executor_sub_query_log_citation_id ON executor_sub_query_log(citation_id) WHERE citation_id IS NOT NULL;

-- ============================================
-- 第六部分：版本记录
-- ============================================

-- 确保 schema_version 表有 UNIQUE 约束
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.table_constraints 
        WHERE constraint_name = 'schema_version_version_key'
    ) THEN
        ALTER TABLE schema_version 
        ADD CONSTRAINT schema_version_version_key UNIQUE (version);
    END IF;
END $$;

-- 插入版本记录（如果不存在）
INSERT INTO schema_version (version, description) 
VALUES 
    ('2.0', '优化索引、添加元数据、级联删除、域名统计表'),
    ('2.1', '添加任务管理表 task_jobs、task_query、executor_sub_query_log'),
    ('2.2', '为 task_jobs 表添加 query_count 字段'),
    ('3.1', '添加任务关联关系：search_records 关联 task_jobs/task_query，executor_sub_query_log 关联 search_records/citations')
ON CONFLICT (version) DO NOTHING;

COMMIT;

-- ============================================
-- 完成后验证
-- ============================================
SELECT 'Complete upgrade to v3.1 completed successfully!' as status;
SELECT version, applied_at, description FROM schema_version ORDER BY applied_at DESC LIMIT 10;

