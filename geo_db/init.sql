CREATE TABLE IF NOT EXISTS search_records (
    id SERIAL PRIMARY KEY,
    keyword TEXT NOT NULL,           -- 用户原始搜索关键词
    platform TEXT NOT NULL,          -- DeepSeek, 豆包等
    prompt_type TEXT,                -- 对比, 建议, 直接查询
    full_answer TEXT,                -- AI 的完整回答
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- 拓展词表：存储 AI 自动生成的搜索关键词
CREATE TABLE IF NOT EXISTS search_queries (
    id SERIAL PRIMARY KEY,
    record_id INTEGER REFERENCES search_records(id),
    query TEXT NOT NULL,             -- AI 拓展的搜索词
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- 引用来源表：存储搜索返回的参考网页
CREATE TABLE IF NOT EXISTS citations (
    id SERIAL PRIMARY KEY,
    record_id INTEGER REFERENCES search_records(id),
    cite_index INTEGER,              -- 引用序号 [1], [2] 等
    url TEXT NOT NULL,               -- 引用链接
    domain TEXT NOT NULL,            -- 提取的域名 (如 zhihu.com)
    title TEXT,                      -- 网页标题
    snippet TEXT,                    -- 内容摘要
    site_name TEXT,                  -- 站点名称
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
