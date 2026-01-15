-- ============================================
-- GEO Monitor - 数据库结构 v3.0
-- 包含：任务管理、查询关联、子查询日志
-- ============================================

-- ============================================
-- 1. 主记录表：存储每次搜索的完整信息
-- ============================================
CREATE TABLE IF NOT EXISTS search_records (
    id SERIAL PRIMARY KEY,
    keyword TEXT NOT NULL,                      -- 用户原始搜索关键词
    platform TEXT NOT NULL,                     -- DeepSeek, 豆包, Kimi 等
    prompt_type TEXT DEFAULT 'default',         -- 对比, 建议, 直接查询
    prompt TEXT,                                -- 完整的提问内容
    full_answer TEXT,                           -- AI 的完整回答
    response_time_ms INTEGER,                   -- 响应时间（毫秒）
    search_status TEXT DEFAULT 'completed',     -- completed, failed, timeout
    error_message TEXT,                         -- 错误信息
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- ============================================
-- 2. 拓展词表：存储 AI 自动生成的搜索关键词
-- ============================================
CREATE TABLE IF NOT EXISTS search_queries (
    id SERIAL PRIMARY KEY,
    record_id INTEGER REFERENCES search_records(id) ON DELETE CASCADE,
    query TEXT NOT NULL,                        -- AI 拓展的搜索词
    query_order INTEGER,                        -- 查询词顺序
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- ============================================
-- 3. 引用来源表：存储搜索返回的参考网页
-- ============================================
CREATE TABLE IF NOT EXISTS citations (
    id SERIAL PRIMARY KEY,
    record_id INTEGER REFERENCES search_records(id) ON DELETE CASCADE,
    cite_index INTEGER,                         -- 引用序号 [1], [2] 等
    url TEXT NOT NULL,                          -- 引用链接
    domain TEXT NOT NULL,                       -- 提取的域名 (如 zhihu.com)
    title TEXT,                                 -- 网页标题
    snippet TEXT,                               -- 内容摘要
    site_name TEXT,                             -- 站点名称
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    -- 唯一约束：防止同一记录重复引用同一 URL
    CONSTRAINT unique_citation UNIQUE (record_id, url)
);

-- ============================================
-- 4. 域名统计表：用于加速聚合分析
-- ============================================
CREATE TABLE IF NOT EXISTS domain_stats (
    id SERIAL PRIMARY KEY,
    domain TEXT UNIQUE NOT NULL,
    total_citations INTEGER DEFAULT 0,          -- 总引用次数
    keyword_coverage INTEGER DEFAULT 0,         -- 覆盖的关键词数
    platforms JSONB DEFAULT '{}',               -- {"DeepSeek": 10, "豆包": 5}
    first_seen TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    last_seen TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- ============================================
-- 5. 任务管理表：存储批量任务信息
-- ============================================
CREATE TABLE IF NOT EXISTS task_jobs (
    id SERIAL PRIMARY KEY,
    keywords JSONB NOT NULL,                    -- 关键词数组
    platforms JSONB NOT NULL,                   -- 平台数组
    query_count INTEGER DEFAULT 1,              -- 查询次数（执行轮数）
    status TEXT DEFAULT 'pending',              -- 任务状态: none, pending, done
    result_data JSONB DEFAULT '{}',             -- 抓取结果数据
    settings JSONB DEFAULT '{}',                -- 任务设置（headless, timeout等）
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- ============================================
-- 6. 任务查询关联表：存储每个任务关联的查询条件
-- ============================================
CREATE TABLE IF NOT EXISTS task_query (
    id SERIAL PRIMARY KEY,
    task_id INTEGER REFERENCES task_jobs(id) ON DELETE CASCADE,
    query TEXT NOT NULL,                        -- 用户输入的原始查询条件
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- ============================================
-- 7. 执行器子查询日志表：存储每次抓取到的子查询和网站信息
-- 注意：sub_query（分词）和网址之间没有直接对应关系，分别存储方便读取和比较
-- ============================================
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

-- ============================================
-- 8. 数据库版本管理表
-- ============================================
CREATE TABLE IF NOT EXISTS schema_version (
    id SERIAL PRIMARY KEY,
    version TEXT UNIQUE NOT NULL,
    applied_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    description TEXT
);

-- ============================================
-- 性能优化索引
-- ============================================

-- 主记录表索引
CREATE INDEX IF NOT EXISTS idx_search_records_keyword ON search_records(keyword);
CREATE INDEX IF NOT EXISTS idx_search_records_platform ON search_records(platform);
CREATE INDEX IF NOT EXISTS idx_search_records_created_at ON search_records(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_search_records_status ON search_records(search_status);

-- 拓展词表索引
CREATE INDEX IF NOT EXISTS idx_search_queries_record ON search_queries(record_id);
CREATE INDEX IF NOT EXISTS idx_search_queries_query ON search_queries(query);

-- 引用表索引
CREATE INDEX IF NOT EXISTS idx_citations_record ON citations(record_id);
CREATE INDEX IF NOT EXISTS idx_citations_domain ON citations(domain);
CREATE INDEX IF NOT EXISTS idx_citations_domain_created ON citations(domain, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_citations_url ON citations(url);

-- 域名统计表索引
CREATE INDEX IF NOT EXISTS idx_domain_stats_domain ON domain_stats(domain);
CREATE INDEX IF NOT EXISTS idx_domain_stats_total ON domain_stats(total_citations DESC);
CREATE INDEX IF NOT EXISTS idx_domain_stats_coverage ON domain_stats(keyword_coverage DESC);

-- 任务管理表索引
CREATE INDEX IF NOT EXISTS idx_task_jobs_status ON task_jobs(status);
CREATE INDEX IF NOT EXISTS idx_task_jobs_created_at ON task_jobs(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_task_jobs_query_count ON task_jobs(query_count);

-- 任务查询关联表索引
CREATE INDEX IF NOT EXISTS idx_task_query_task_id ON task_query(task_id);
CREATE INDEX IF NOT EXISTS idx_task_query_query ON task_query(query);
CREATE INDEX IF NOT EXISTS idx_task_query_created_at ON task_query(created_at DESC);

-- 执行器子查询日志表索引
CREATE INDEX IF NOT EXISTS idx_executor_sub_query_log_task_query_id ON executor_sub_query_log(task_query_id);
CREATE INDEX IF NOT EXISTS idx_executor_sub_query_log_url ON executor_sub_query_log(url) WHERE url IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_executor_sub_query_log_domain ON executor_sub_query_log(domain) WHERE domain IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_executor_sub_query_log_created_at ON executor_sub_query_log(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_executor_sub_query_log_sub_query ON executor_sub_query_log(sub_query) WHERE sub_query IS NOT NULL;

-- ============================================
-- 触发器：自动更新 updated_at
-- ============================================

CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER update_search_records_updated_at BEFORE UPDATE ON search_records
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_domain_stats_updated_at BEFORE UPDATE ON domain_stats
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_task_jobs_updated_at BEFORE UPDATE ON task_jobs
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- ============================================
-- 版本记录
-- ============================================

INSERT INTO schema_version (version, description) 
VALUES ('3.0', '添加任务管理、查询关联、子查询日志表，优化数据结构')
ON CONFLICT DO NOTHING;

