-- ============================================
-- 迁移脚本：006_add_users_and_auth_tables.sql
-- 描述：添加用户认证相关表（users表、auth_tokens表）
-- 创建日期：2025-01-XX
-- ============================================

BEGIN;

-- ============================================
-- 1. 用户表（users）
-- 用于存储系统用户信息，支持客户端登录认证
-- ============================================
CREATE TABLE IF NOT EXISTS users (
    id SERIAL PRIMARY KEY,
    username VARCHAR(100) UNIQUE NOT NULL,          -- 用户名（唯一）
    password_hash TEXT NOT NULL,                     -- 加密后的密码（bcrypt）
    email VARCHAR(255),                              -- 邮箱（可选）
    full_name VARCHAR(255),                          -- 全名（可选）
    is_active BOOLEAN DEFAULT TRUE,                  -- 是否激活
    is_admin BOOLEAN DEFAULT FALSE,                  -- 是否管理员
    last_login_at TIMESTAMP,                         -- 最后登录时间
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    -- 检查约束
    CONSTRAINT users_username_length_check CHECK (char_length(username) >= 3 AND char_length(username) <= 100)
);

-- 索引
CREATE INDEX IF NOT EXISTS idx_users_username ON users(username);
CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);
CREATE INDEX IF NOT EXISTS idx_users_is_active ON users(is_active);

-- ============================================
-- 2. 认证Token表（auth_tokens）
-- 用于存储客户端登录token信息（可选，也可以只使用JWT）
-- 如果使用纯JWT方案，这个表可以用于token撤销和黑名单管理
-- ============================================
CREATE TABLE IF NOT EXISTS auth_tokens (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL,                        -- 关联用户ID
    token TEXT NOT NULL,                             -- JWT token或token哈希
    token_hash TEXT NOT NULL,                        -- token的哈希值（用于快速查找）
    expires_at TIMESTAMP NOT NULL,                   -- token过期时间
    device_info TEXT,                                -- 设备信息（可选）
    is_revoked BOOLEAN DEFAULT FALSE,                -- 是否已撤销
    revoked_at TIMESTAMP,                            -- 撤销时间
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    -- 外键约束
    CONSTRAINT auth_tokens_user_id_fkey 
        FOREIGN KEY (user_id) 
        REFERENCES users(id) 
        ON DELETE CASCADE,
    
    -- 唯一约束：确保同一用户不会重复相同的token
    CONSTRAINT auth_tokens_token_hash_unique UNIQUE (token_hash)
);

-- 索引
CREATE INDEX IF NOT EXISTS idx_auth_tokens_user_id ON auth_tokens(user_id);
CREATE INDEX IF NOT EXISTS idx_auth_tokens_token_hash ON auth_tokens(token_hash);
CREATE INDEX IF NOT EXISTS idx_auth_tokens_expires_at ON auth_tokens(expires_at);
CREATE INDEX IF NOT EXISTS idx_auth_tokens_is_revoked ON auth_tokens(is_revoked);

-- ============================================
-- 3. 触发器：自动更新 updated_at 字段
-- ============================================
CREATE TRIGGER update_users_updated_at
    BEFORE UPDATE ON users
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- ============================================
-- 4. 函数：清理过期的token
-- ============================================
CREATE OR REPLACE FUNCTION cleanup_expired_tokens()
RETURNS void AS $$
BEGIN
    DELETE FROM auth_tokens
    WHERE expires_at < CURRENT_TIMESTAMP
       OR (is_revoked = TRUE AND revoked_at < CURRENT_TIMESTAMP - INTERVAL '30 days');
END;
$$ LANGUAGE plpgsql;

-- ============================================
-- 5. 视图：用户登录统计
-- ============================================
CREATE OR REPLACE VIEW user_login_stats AS
SELECT 
    u.id,
    u.username,
    u.email,
    u.is_active,
    u.last_login_at,
    COUNT(DISTINCT at.id) as active_tokens_count,
    MAX(at.created_at) as latest_token_created_at
FROM users u
LEFT JOIN auth_tokens at ON u.id = at.user_id 
    AND at.is_revoked = FALSE 
    AND at.expires_at > CURRENT_TIMESTAMP
GROUP BY u.id, u.username, u.email, u.is_active, u.last_login_at;

COMMIT;

-- ============================================
-- 迁移完成
-- ============================================
-- 注意：创建用户需要使用密码加密工具（bcrypt）
-- 示例（Python）：
-- import bcrypt
-- password_hash = bcrypt.hashpw('password'.encode('utf-8'), bcrypt.gensalt()).decode('utf-8')
-- INSERT INTO users (username, password_hash) VALUES ('admin', password_hash);
