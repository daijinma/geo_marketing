-- ============================================
-- GEO Monitor - 优化后的数据库结构 v2.0
-- 包含：索引优化、约束增强、元数据扩展
-- ============================================

-- 主记录表：存储每次搜索的完整信息
CREATE TABLE IF NOT EXISTS search_records (
    id SERIAL PRIMARY KEY,
    keyword TEXT NOT NULL,                      -- 用户原始搜索关键词
    platform TEXT NOT NULL,                     -- DeepSeek, 豆包, Kimi 等
    prompt_type TEXT DEFAULT 'default',         -- 对比, 建议, 直接查询
    prompt TEXT,                                -- 【新增】完整的提问内容
    full_answer TEXT,                           -- AI 的完整回答
    response_time_ms INTEGER,                   -- 【新增】响应时间（毫秒）
    search_status TEXT DEFAULT 'completed',     -- 【新增】completed, failed, timeout
    error_message TEXT,                         -- 【新增】错误信息
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- 拓展词表：存储 AI 自动生成的搜索关键词
CREATE TABLE IF NOT EXISTS search_queries (
    id SERIAL PRIMARY KEY,
    record_id INTEGER REFERENCES search_records(id) ON DELETE CASCADE,
    query TEXT NOT NULL,                        -- AI 拓展的搜索词
    query_order INTEGER,                        -- 【新增】查询词顺序
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- 引用来源表：存储搜索返回的参考网页
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
    
    -- 【新增】唯一约束：防止同一记录重复引用同一 URL
    CONSTRAINT unique_citation UNIQUE (record_id, url)
);

-- 域名统计表：用于加速聚合分析
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

-- ============================================
-- 数据库版本管理
-- ============================================

CREATE TABLE IF NOT EXISTS schema_version (
    id SERIAL PRIMARY KEY,
    version TEXT NOT NULL,
    applied_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    description TEXT
);

INSERT INTO schema_version (version, description) 
VALUES ('2.0', '优化索引、添加元数据、级联删除、域名统计表')
ON CONFLICT DO NOTHING;
