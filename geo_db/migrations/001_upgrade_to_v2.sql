-- ============================================
-- 数据库升级脚本：从 v1.0 到 v2.0
-- ============================================

BEGIN;

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
ALTER TABLE search_queries 
DROP CONSTRAINT IF EXISTS search_queries_record_id_fkey,
ADD CONSTRAINT search_queries_record_id_fkey 
    FOREIGN KEY (record_id) 
    REFERENCES search_records(id) 
    ON DELETE CASCADE;

-- 3. 增强引用表
ALTER TABLE citations 
DROP CONSTRAINT IF EXISTS citations_record_id_fkey,
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

ALTER TABLE citations 
ADD CONSTRAINT unique_citation UNIQUE (record_id, url);

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

-- 6. 添加所有索引
CREATE INDEX IF NOT EXISTS idx_search_records_keyword ON search_records(keyword);
CREATE INDEX IF NOT EXISTS idx_search_records_platform ON search_records(platform);
CREATE INDEX IF NOT EXISTS idx_search_records_created_at ON search_records(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_search_records_status ON search_records(search_status);

CREATE INDEX IF NOT EXISTS idx_search_queries_record ON search_queries(record_id);
CREATE INDEX IF NOT EXISTS idx_search_queries_query ON search_queries(query);

CREATE INDEX IF NOT EXISTS idx_citations_record ON citations(record_id);
CREATE INDEX IF NOT EXISTS idx_citations_domain ON citations(domain);
CREATE INDEX IF NOT EXISTS idx_citations_domain_created ON citations(domain, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_citations_url ON citations(url);

CREATE INDEX IF NOT EXISTS idx_domain_stats_domain ON domain_stats(domain);
CREATE INDEX IF NOT EXISTS idx_domain_stats_total ON domain_stats(total_citations DESC);
CREATE INDEX IF NOT EXISTS idx_domain_stats_coverage ON domain_stats(keyword_coverage DESC);

-- 7. 创建触发器
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS update_search_records_updated_at ON search_records;
CREATE TRIGGER update_search_records_updated_at BEFORE UPDATE ON search_records
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

DROP TRIGGER IF EXISTS update_domain_stats_updated_at ON domain_stats;
CREATE TRIGGER update_domain_stats_updated_at BEFORE UPDATE ON domain_stats
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- 8. 版本记录
CREATE TABLE IF NOT EXISTS schema_version (
    id SERIAL PRIMARY KEY,
    version TEXT NOT NULL,
    applied_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    description TEXT
);

INSERT INTO schema_version (version, description) 
VALUES ('2.0', '优化索引、添加元数据、级联删除、域名统计表');

COMMIT;

-- 完成后验证
SELECT 'Migration completed successfully!' as status;
SELECT version, applied_at, description FROM schema_version ORDER BY applied_at DESC LIMIT 5;
