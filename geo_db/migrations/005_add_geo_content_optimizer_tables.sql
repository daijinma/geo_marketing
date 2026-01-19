-- ============================================
-- 迁移脚本：005_add_geo_content_optimizer_tables.sql
-- 描述：添加 GEO 内容优化系统相关表
-- 创建日期：2025-01-16
-- ============================================

BEGIN;

-- ============================================
-- 1. 话题映射表（topic_maps）
-- 用于存储话题邻近性映射结果，支持语义足迹扩展
-- ============================================
CREATE TABLE IF NOT EXISTS topic_maps (
    id SERIAL PRIMARY KEY,
    core_keyword VARCHAR(255) NOT NULL,
    related_topic VARCHAR(255) NOT NULL,
    topic_type VARCHAR(50), -- 'user_intent' / 'comparison' / 'process' / 'data'
    user_intent TEXT,
    keywords TEXT[], -- 关键词扩展数组
    content_type VARCHAR(50), -- 'data' / 'process' / 'comparison' / 'standard'
    priority INTEGER DEFAULT 5, -- 1-10, 越高优先级越高
    content_url TEXT,
    status VARCHAR(20) DEFAULT 'pending', -- 'pending' / 'created' / 'optimized'
    industry VARCHAR(100), -- 行业领域
    notes TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    -- 唯一约束
    CONSTRAINT topic_maps_core_keyword_unique UNIQUE (core_keyword, related_topic),
    
    -- 检查约束
    CONSTRAINT topic_maps_priority_check CHECK (priority >= 1 AND priority <= 10),
    CONSTRAINT topic_maps_status_check CHECK (status IN ('pending', 'created', 'optimized'))
);

-- 索引
CREATE INDEX IF NOT EXISTS idx_topic_maps_core_keyword ON topic_maps(core_keyword);
CREATE INDEX IF NOT EXISTS idx_topic_maps_status ON topic_maps(status);
CREATE INDEX IF NOT EXISTS idx_topic_maps_priority ON topic_maps(priority DESC);
CREATE INDEX IF NOT EXISTS idx_topic_maps_industry ON topic_maps(industry);

-- ============================================
-- 2. 事实数据源表（fact_sources）
-- 用于存储统计数据、引用文献和独特见解，支持事实密度提升
-- ============================================
CREATE TABLE IF NOT EXISTS fact_sources (
    id SERIAL PRIMARY KEY,
    category VARCHAR(50) NOT NULL, -- 'statistics' / 'citation' / 'insight'
    source_type VARCHAR(100), -- 来源类型：官方统计/行业报告/企业数据/用户调研/国家标准/行业标准/学术论文/专家观点
    title VARCHAR(500),
    content TEXT,
    data_value VARCHAR(500), -- 数据值
    data_unit VARCHAR(50), -- 数据单位
    source_url TEXT,
    author_org VARCHAR(255), -- 作者/机构
    publish_date DATE,
    validity_period INTERVAL, -- 有效期
    expiration_date DATE, -- 过期日期
    tags JSONB, -- 标签（JSON 格式）
    verification_status VARCHAR(20) DEFAULT 'pending', -- 'pending' / 'verified' / 'expired' / 'invalid'
    quality_score INTEGER, -- 质量评分 0-100
    collection_method VARCHAR(50), -- 'manual' / 'api' / 'crawler'
    update_frequency VARCHAR(50), -- 'monthly' / 'quarterly' / 'yearly' / 'on_demand'
    industry VARCHAR(100), -- 行业领域
    notes TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    -- 检查约束
    CONSTRAINT fact_sources_category_check CHECK (category IN ('statistics', 'citation', 'insight')),
    CONSTRAINT fact_sources_verification_status_check CHECK (verification_status IN ('pending', 'verified', 'expired', 'invalid')),
    CONSTRAINT fact_sources_quality_score_check CHECK (quality_score >= 0 AND quality_score <= 100)
);

-- 索引
CREATE INDEX IF NOT EXISTS idx_fact_sources_category ON fact_sources(category);
CREATE INDEX IF NOT EXISTS idx_fact_sources_verification_status ON fact_sources(verification_status);
CREATE INDEX IF NOT EXISTS idx_fact_sources_publish_date ON fact_sources(publish_date DESC);
CREATE INDEX IF NOT EXISTS idx_fact_sources_expiration_date ON fact_sources(expiration_date);
CREATE INDEX IF NOT EXISTS idx_fact_sources_industry ON fact_sources(industry);
CREATE INDEX IF NOT EXISTS idx_fact_sources_tags ON fact_sources USING GIN(tags); -- GIN 索引支持 JSONB 查询

-- ============================================
-- 3. Schema 实施记录表（schema_implementations）
-- 用于记录 Schema.org 标记的实施情况，支持结构化数据实施
-- ============================================
CREATE TABLE IF NOT EXISTS schema_implementations (
    id SERIAL PRIMARY KEY,
    page_url TEXT NOT NULL,
    page_title VARCHAR(500),
    schema_type VARCHAR(100) NOT NULL, -- 'Organization' / 'Service' / 'FAQPage' / 'Article' / 'LocalBusiness' / etc.
    schema_json JSONB NOT NULL, -- Schema 的 JSON-LD 格式数据
    validation_status VARCHAR(20), -- 'valid' / 'invalid' / 'pending' / 'warning'
    validation_errors JSONB, -- 验证错误信息（JSON 格式）
    validation_date TIMESTAMP, -- 最后验证时间
    priority VARCHAR(20), -- 'high' / 'medium' / 'low'
    implementation_date DATE, -- 实施日期
    industry VARCHAR(100), -- 行业领域
    notes TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    -- 检查约束
    CONSTRAINT schema_implementations_validation_status_check CHECK (validation_status IN ('valid', 'invalid', 'pending', 'warning')),
    CONSTRAINT schema_implementations_priority_check CHECK (priority IN ('high', 'medium', 'low'))
);

-- 索引
CREATE INDEX IF NOT EXISTS idx_schema_implementations_page_url ON schema_implementations(page_url);
CREATE INDEX IF NOT EXISTS idx_schema_implementations_schema_type ON schema_implementations(schema_type);
CREATE INDEX IF NOT EXISTS idx_schema_implementations_validation_status ON schema_implementations(validation_status);
CREATE INDEX IF NOT EXISTS idx_schema_implementations_priority ON schema_implementations(priority);
CREATE INDEX IF NOT EXISTS idx_schema_implementations_schema_json ON schema_implementations USING GIN(schema_json); -- GIN 索引支持 JSONB 查询

-- ============================================
-- 4. 更新任务表（update_tasks）
-- 用于管理内容更新任务，支持基于监控反馈的更新机制
-- ============================================
CREATE TABLE IF NOT EXISTS update_tasks (
    id SERIAL PRIMARY KEY,
    page_url TEXT NOT NULL,
    page_title VARCHAR(500),
    task_type VARCHAR(50), -- 'data_refresh' / 'content_enhance' / 'schema_update' / 'topic_expansion'
    priority INTEGER DEFAULT 5, -- 1-10, 越高优先级越高
    status VARCHAR(20) DEFAULT 'pending', -- 'pending' / 'in_progress' / 'completed' / 'cancelled'
    trigger_reason TEXT, -- 触发原因
    trigger_metric VARCHAR(100), -- 触发指标：citation_drop / data_expired / new_topic / competitor_overtake
    trigger_value NUMERIC, -- 触发指标值
    assigned_to VARCHAR(100), -- 分配给谁
    due_date DATE, -- 截止日期
    completed_at TIMESTAMP, -- 完成时间
    completion_notes TEXT, -- 完成备注
    industry VARCHAR(100), -- 行业领域
    notes TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    -- 检查约束
    CONSTRAINT update_tasks_status_check CHECK (status IN ('pending', 'in_progress', 'completed', 'cancelled')),
    CONSTRAINT update_tasks_task_type_check CHECK (task_type IN ('data_refresh', 'content_enhance', 'schema_update', 'topic_expansion')),
    CONSTRAINT update_tasks_priority_check CHECK (priority >= 1 AND priority <= 10)
);

-- 索引
CREATE INDEX IF NOT EXISTS idx_update_tasks_page_url ON update_tasks(page_url);
CREATE INDEX IF NOT EXISTS idx_update_tasks_status ON update_tasks(status);
CREATE INDEX IF NOT EXISTS idx_update_tasks_priority ON update_tasks(priority DESC);
CREATE INDEX IF NOT EXISTS idx_update_tasks_task_type ON update_tasks(task_type);
CREATE INDEX IF NOT EXISTS idx_update_tasks_due_date ON update_tasks(due_date);

-- ============================================
-- 5. 内容质量评分表（content_quality_scores）
-- 用于记录页面内容的质量评分，支持事实密度评分系统
-- ============================================
CREATE TABLE IF NOT EXISTS content_quality_scores (
    id SERIAL PRIMARY KEY,
    page_url TEXT NOT NULL,
    page_title VARCHAR(500),
    score INTEGER NOT NULL, -- 总得分 0-100
    statistics_count INTEGER DEFAULT 0, -- 统计数据数量
    statistics_score INTEGER DEFAULT 0, -- 统计数据得分（0-30）
    citation_count INTEGER DEFAULT 0, -- 引用来源数量
    citation_score INTEGER DEFAULT 0, -- 引用来源得分（0-30）
    authority_score INTEGER DEFAULT 0, -- 来源权威性得分（0-20）
    timeliness_score INTEGER DEFAULT 0, -- 数据时效性得分（0-20）
    scoring_date DATE NOT NULL, -- 评分日期
    previous_score INTEGER, -- 上次评分
    score_change INTEGER, -- 得分变化
    industry VARCHAR(100), -- 行业领域
    notes TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    -- 检查约束
    CONSTRAINT content_quality_scores_score_check CHECK (score >= 0 AND score <= 100),
    CONSTRAINT content_quality_scores_statistics_score_check CHECK (statistics_score >= 0 AND statistics_score <= 30),
    CONSTRAINT content_quality_scores_citation_score_check CHECK (citation_score >= 0 AND citation_score <= 30),
    CONSTRAINT content_quality_scores_authority_score_check CHECK (authority_score >= 0 AND authority_score <= 20),
    CONSTRAINT content_quality_scores_timeliness_score_check CHECK (timeliness_score >= 0 AND timeliness_score <= 20),
    
    -- 唯一约束：每个页面每天只能有一条评分记录
    CONSTRAINT content_quality_scores_page_date_unique UNIQUE (page_url, scoring_date)
);

-- 索引
CREATE INDEX IF NOT EXISTS idx_content_quality_scores_page_url ON content_quality_scores(page_url);
CREATE INDEX IF NOT EXISTS idx_content_quality_scores_scoring_date ON content_quality_scores(scoring_date DESC);
CREATE INDEX IF NOT EXISTS idx_content_quality_scores_score ON content_quality_scores(score DESC);

-- ============================================
-- 6. 触发器函数：自动更新 updated_at 字段
-- ============================================
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- 为每个表创建触发器
CREATE TRIGGER update_topic_maps_updated_at
    BEFORE UPDATE ON topic_maps
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_fact_sources_updated_at
    BEFORE UPDATE ON fact_sources
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_schema_implementations_updated_at
    BEFORE UPDATE ON schema_implementations
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_update_tasks_updated_at
    BEFORE UPDATE ON update_tasks
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_content_quality_scores_updated_at
    BEFORE UPDATE ON content_quality_scores
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- ============================================
-- 7. 函数：自动计算过期日期
-- ============================================
CREATE OR REPLACE FUNCTION calculate_expiration_date()
RETURNS TRIGGER AS $$
BEGIN
    IF NEW.validity_period IS NOT NULL AND NEW.publish_date IS NOT NULL THEN
        NEW.expiration_date = NEW.publish_date + NEW.validity_period;
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER calculate_fact_sources_expiration_date
    BEFORE INSERT OR UPDATE ON fact_sources
    FOR EACH ROW
    EXECUTE FUNCTION calculate_expiration_date();

-- ============================================
-- 8. 函数：自动标记过期数据源
-- ============================================
CREATE OR REPLACE FUNCTION mark_expired_fact_sources()
RETURNS void AS $$
BEGIN
    UPDATE fact_sources
    SET verification_status = 'expired'
    WHERE expiration_date < CURRENT_DATE
      AND verification_status = 'verified';
END;
$$ LANGUAGE plpgsql;

-- ============================================
-- 9. 视图：话题覆盖统计
-- ============================================
CREATE OR REPLACE VIEW topic_coverage_stats AS
SELECT 
    core_keyword,
    COUNT(*) as total_topics,
    COUNT(CASE WHEN status = 'completed' THEN 1 END) as completed_topics,
    COUNT(CASE WHEN status = 'pending' THEN 1 END) as pending_topics,
    COUNT(CASE WHEN status = 'created' THEN 1 END) as created_topics,
    ROUND(COUNT(CASE WHEN status = 'completed' THEN 1 END) * 100.0 / COUNT(*), 2) as completion_rate
FROM topic_maps
GROUP BY core_keyword;

-- ============================================
-- 10. 视图：数据源质量统计
-- ============================================
CREATE OR REPLACE VIEW fact_sources_quality_stats AS
SELECT 
    category,
    source_type,
    COUNT(*) as total_sources,
    COUNT(CASE WHEN verification_status = 'verified' THEN 1 END) as verified_sources,
    COUNT(CASE WHEN verification_status = 'expired' THEN 1 END) as expired_sources,
    AVG(quality_score) as avg_quality_score,
    MIN(quality_score) as min_quality_score,
    MAX(quality_score) as max_quality_score
FROM fact_sources
GROUP BY category, source_type;

-- ============================================
-- 11. 视图：Schema 实施统计
-- ============================================
CREATE OR REPLACE VIEW schema_implementation_stats AS
SELECT 
    schema_type,
    COUNT(*) as total_implementations,
    COUNT(CASE WHEN validation_status = 'valid' THEN 1 END) as valid_count,
    COUNT(CASE WHEN validation_status = 'invalid' THEN 1 END) as invalid_count,
    COUNT(CASE WHEN validation_status = 'pending' THEN 1 END) as pending_count,
    ROUND(COUNT(CASE WHEN validation_status = 'valid' THEN 1 END) * 100.0 / COUNT(*), 2) as validation_rate
FROM schema_implementations
GROUP BY schema_type;

-- ============================================
-- 12. 视图：更新任务统计
-- ============================================
CREATE OR REPLACE VIEW update_tasks_stats AS
SELECT 
    task_type,
    status,
    COUNT(*) as task_count,
    AVG(priority) as avg_priority,
    COUNT(CASE WHEN due_date < CURRENT_DATE AND status != 'completed' THEN 1 END) as overdue_count
FROM update_tasks
GROUP BY task_type, status;

COMMIT;

-- ============================================
-- 迁移完成
-- ============================================

