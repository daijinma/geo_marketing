use anyhow::Result;
use dirs;
use rusqlite::{params, Connection, Result as SqliteResult};
use std::path::PathBuf;
use std::sync::Mutex;

static DB_CONNECTION: Mutex<Option<Connection>> = Mutex::new(None);

/// 获取数据库路径
fn get_db_path() -> Result<PathBuf> {
    let home_dir = dirs::home_dir().ok_or_else(|| anyhow::anyhow!("无法获取用户主目录"))?;
    let app_dir = home_dir.join(".geo_client");
    
    // 确保目录存在
    std::fs::create_dir_all(&app_dir)?;
    
    Ok(app_dir.join("cache.db"))
}

/// 初始化数据库（确保表存在）
pub fn init_db() -> Result<()> {
    let db_path = get_db_path()?;
    let conn = Connection::open(&db_path)?;
    
    // 创建表
    create_tables(&conn)?;
    
    // 更新数据库版本（如果需要迁移）
    update_schema_version(&conn)?;
    
    Ok(())
}

/// 获取数据库连接
fn get_connection() -> Result<Connection> {
    // 确保数据库已初始化
    if DB_CONNECTION.lock().unwrap().is_none() {
        init_db()?;
    }
    
    // 每次使用新连接（rusqlite的连接不是线程安全的，需要在同一个线程使用）
    let db_path = get_db_path()?;
    let conn = Connection::open(&db_path)?;
    Ok(conn)
}

/// 创建所有表
fn create_tables(conn: &Connection) -> Result<()> {
    // auth 表：存储认证token
    conn.execute(
        "CREATE TABLE IF NOT EXISTS auth (
            id INTEGER PRIMARY KEY AUTOINCREMENT,
            token TEXT NOT NULL,
            expires_at TEXT NOT NULL,
            created_at TEXT DEFAULT CURRENT_TIMESTAMP,
            updated_at TEXT DEFAULT CURRENT_TIMESTAMP
        )",
        [],
    )?;
    
    // tasks 表：本地任务数据
    conn.execute(
        "CREATE TABLE IF NOT EXISTS tasks (
            id INTEGER PRIMARY KEY AUTOINCREMENT,
            task_id INTEGER,
            keywords TEXT NOT NULL,
            platforms TEXT NOT NULL,
            query_count INTEGER NOT NULL,
            status TEXT NOT NULL,
            result_data TEXT,
            created_at TEXT DEFAULT CURRENT_TIMESTAMP,
            updated_at TEXT DEFAULT CURRENT_TIMESTAMP,
            UNIQUE(task_id)
        )",
        [],
    )?;
    
    // 添加索引
    conn.execute(
        "CREATE INDEX IF NOT EXISTS idx_tasks_status ON tasks(status)",
        [],
    )?;
    
    // login_status 表：平台登录状态
    conn.execute(
        "CREATE TABLE IF NOT EXISTS login_status (
            id INTEGER PRIMARY KEY AUTOINCREMENT,
            platform_type TEXT NOT NULL,
            platform_name TEXT NOT NULL,
            is_logged_in INTEGER NOT NULL DEFAULT 0,
            last_check_at TEXT,
            updated_at TEXT DEFAULT CURRENT_TIMESTAMP,
            UNIQUE(platform_name)
        )",
        [],
    )?;
    
    // 添加索引
    conn.execute(
        "CREATE INDEX IF NOT EXISTS idx_login_status_platform ON login_status(platform_name)",
        [],
    )?;
    
    // settings 表：用户设置
    conn.execute(
        "CREATE TABLE IF NOT EXISTS settings (
            key TEXT PRIMARY KEY,
            value TEXT NOT NULL
        )",
        [],
    )?;
    
    // logs 表：日志记录
    conn.execute(
        "CREATE TABLE IF NOT EXISTS logs (
            id INTEGER PRIMARY KEY AUTOINCREMENT,
            timestamp TEXT NOT NULL,
            level TEXT NOT NULL,
            module TEXT,
            message TEXT NOT NULL,
            error_detail TEXT,
            task_id INTEGER,
            created_at TEXT DEFAULT CURRENT_TIMESTAMP
        )",
        [],
    )?;
    
    // 添加索引
    conn.execute(
        "CREATE INDEX IF NOT EXISTS idx_logs_timestamp ON logs(timestamp)",
        [],
    )?;
    conn.execute(
        "CREATE INDEX IF NOT EXISTS idx_logs_level ON logs(level)",
        [],
    )?;
    conn.execute(
        "CREATE INDEX IF NOT EXISTS idx_logs_module ON logs(module)",
        [],
    )?;
    conn.execute(
        "CREATE INDEX IF NOT EXISTS idx_logs_task_id ON logs(task_id)",
        [],
    )?;
    
    Ok(())
}

/// 更新数据库schema版本
fn update_schema_version(conn: &Connection) -> Result<()> {
    // 创建schema_version表（如果不存在）
    conn.execute(
        "CREATE TABLE IF NOT EXISTS schema_version (
            version INTEGER PRIMARY KEY
        )",
        [],
    )?;
    
    // 检查当前版本
    let current_version: i32 = conn
        .query_row(
            "SELECT version FROM schema_version LIMIT 1",
            [],
            |row| row.get(0),
        )
        .unwrap_or(0);
    
    const LATEST_VERSION: i32 = 1;
    
    // 如果需要更新，执行迁移
    if current_version < LATEST_VERSION {
        // 迁移逻辑（当前版本1，暂时不需要迁移）
        conn.execute(
            "INSERT OR REPLACE INTO schema_version (version) VALUES (?1)",
            params![LATEST_VERSION],
        )?;
    }
    
    Ok(())
}

/// 保存认证token
pub fn save_auth_token(token: &str, expires_at: &str) -> Result<()> {
    let conn = get_connection()?;
    
    // 先删除旧的token（只保留一个）
    conn.execute("DELETE FROM auth", [])?;
    
    // 插入新token
    conn.execute(
        "INSERT INTO auth (token, expires_at, updated_at) VALUES (?1, ?2, datetime('now'))",
        params![token, expires_at],
    )?;
    
    Ok(())
}

/// 获取认证token
pub fn get_auth_token() -> Result<Option<(String, String)>> {
    let conn = get_connection()?;
    
    let mut stmt = conn.prepare("SELECT token, expires_at FROM auth ORDER BY id DESC LIMIT 1")?;
    let mut rows = stmt.query_map([], |row| {
        Ok((row.get::<_, String>(0)?, row.get::<_, String>(1)?))
    })?;
    
    if let Some(row) = rows.next() {
        Ok(Some(row?))
    } else {
        Ok(None)
    }
}

/// 删除认证token
pub fn delete_auth_token() -> Result<()> {
    let conn = get_connection()?;
    conn.execute("DELETE FROM auth", [])?;
    Ok(())
}

/// 检查token是否过期
pub fn is_token_expired() -> Result<bool> {
    if let Some((_, expires_at_str)) = get_auth_token()? {
        // 尝试解析RFC3339格式
        let expires_at = match chrono::DateTime::parse_from_rfc3339(&expires_at_str) {
            Ok(dt) => dt.with_timezone(&chrono::Utc),
            Err(_) => {
                // 尝试ISO8601格式
                match chrono::DateTime::parse_from_rfc2822(&expires_at_str) {
                    Ok(dt) => dt.with_timezone(&chrono::Utc),
                    Err(_) => {
                        // 尝试自定义格式
                        chrono::NaiveDateTime::parse_from_str(&expires_at_str, "%Y-%m-%d %H:%M:%S")
                            .or_else(|_| {
                                chrono::NaiveDateTime::parse_from_str(&expires_at_str, "%Y-%m-%dT%H:%M:%S%.fZ")
                            })?
                            .and_utc()
                    }
                }
            }
        };
        
        Ok(chrono::Utc::now() > expires_at)
    } else {
        Ok(true) // 没有token视为过期
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    
    #[test]
    fn test_db_path() {
        let path = get_db_path().unwrap();
        assert!(path.to_string_lossy().contains(".geo_client"));
    }
}
