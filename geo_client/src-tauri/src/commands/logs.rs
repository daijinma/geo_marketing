use crate::storage::database;
use anyhow::Result;
use rusqlite::{params, ToSql};
use serde::{Deserialize, Serialize};
use tauri::command;

#[derive(Debug, Serialize, Deserialize)]
pub struct LogEntry {
    pub id: i64,
    pub timestamp: String,
    pub level: String,
    pub module: Option<String>,
    pub message: String,
    pub error_detail: Option<String>,
    pub task_id: Option<i64>,
}

/// 记录日志
pub fn log_message(
    level: &str,
    module: Option<&str>,
    message: &str,
    error_detail: Option<&str>,
    task_id: Option<i64>,
) -> Result<()> {
    let conn = database::get_connection()?;
    
    conn.execute(
        "INSERT INTO logs (timestamp, level, module, message, error_detail, task_id) 
         VALUES (datetime('now'), ?1, ?2, ?3, ?4, ?5)",
        params![
            level,
            module,
            message,
            error_detail,
            task_id
        ],
    )?;
    
    Ok(())
}

/// 获取日志列表
#[command]
pub fn get_logs(
    level: Option<String>,
    module: Option<String>,
    start_time: Option<String>,
    end_time: Option<String>,
    keyword: Option<String>,
    limit: i32,
    offset: i32,
) -> Result<Vec<LogEntry>, String> {
    let conn = database::get_connection()
        .map_err(|e| format!("获取数据库连接失败: {}", e))?;

    // 构建查询和参数
    let mut conditions = Vec::new();
    let mut query_params: Vec<Box<dyn ToSql>> = Vec::new();

    if let Some(ref lvl) = level {
        conditions.push("level = ?");
        query_params.push(Box::new(lvl.as_str()));
    }

    if let Some(ref mod_name) = module {
        conditions.push("module = ?");
        query_params.push(Box::new(mod_name.as_str()));
    }

    if let Some(ref start) = start_time {
        conditions.push("timestamp >= ?");
        query_params.push(Box::new(start.as_str()));
    }

    if let Some(ref end) = end_time {
        conditions.push("timestamp <= ?");
        query_params.push(Box::new(end.as_str()));
    }

    if let Some(ref kw) = keyword {
        conditions.push("message LIKE ?");
        query_params.push(Box::new(format!("%{}%", kw)));
    }

    let where_clause = if conditions.is_empty() {
        String::new()
    } else {
        format!(" WHERE {}", conditions.join(" AND "))
    };

    query_params.push(Box::new(limit));
    query_params.push(Box::new(offset));

    let query = format!(
        "SELECT id, timestamp, level, module, message, error_detail, task_id 
         FROM logs{} 
         ORDER BY timestamp DESC 
         LIMIT ? OFFSET ?",
        where_clause
    );

    let mut stmt = conn
        .prepare(&query)
        .map_err(|e| format!("准备查询失败: {}", e))?;

    // 使用 rusqlite 的 params_from_iter 来处理动态参数
    use rusqlite::params_from_iter;
    
    let rows = stmt
        .query_map(params_from_iter(query_params.iter().map(|p| p.as_ref())), |row| {
            Ok(LogEntry {
                id: row.get(0)?,
                timestamp: row.get(1)?,
                level: row.get(2)?,
                module: row.get(3)?,
                message: row.get(4)?,
                error_detail: row.get(5)?,
                task_id: row.get(6)?,
            })
        })
        .map_err(|e| format!("查询失败: {}", e))?;

    let mut result = Vec::new();
    for row in rows {
        result.push(row.map_err(|e| format!("解析行失败: {}", e))?);
    }

    Ok(result)
}

/// 获取日志统计
#[command]
pub fn get_log_stats() -> Result<LogStats, String> {
    let conn = database::get_connection()
        .map_err(|e| format!("获取数据库连接失败: {}", e))?;

    let mut stats = LogStats {
        total: 0,
        debug: 0,
        info: 0,
        warn: 0,
        error: 0,
        today_count: 0,
    };

    // 获取总数
    let total: i64 = conn
        .query_row("SELECT COUNT(*) FROM logs", [], |row| row.get(0))
        .map_err(|e| format!("查询总数失败: {}", e))?;
    stats.total = total as i32;

    // 按级别统计
    for level in &["DEBUG", "INFO", "WARN", "ERROR"] {
        let count: i64 = conn
            .query_row(
                "SELECT COUNT(*) FROM logs WHERE level = ?1",
                params![level],
                |row| row.get(0),
            )
            .unwrap_or(0);
        
        match *level {
            "DEBUG" => stats.debug = count as i32,
            "INFO" => stats.info = count as i32,
            "WARN" => stats.warn = count as i32,
            "ERROR" => stats.error = count as i32,
            _ => {}
        }
    }

    // 今日日志数
    let today_count: i64 = conn
        .query_row(
            "SELECT COUNT(*) FROM logs WHERE date(timestamp) = date('now')",
            [],
            |row| row.get(0),
        )
        .unwrap_or(0);
    stats.today_count = today_count as i32;

    Ok(stats)
}

#[derive(Debug, Serialize, Deserialize)]
pub struct LogStats {
    pub total: i32,
    pub debug: i32,
    pub info: i32,
    pub warn: i32,
    pub error: i32,
    pub today_count: i32,
}

/// 清空日志（保留最近N条）
#[command]
pub fn clear_logs(keep_count: i32) -> Result<(), String> {
    let conn = database::get_connection()
        .map_err(|e| format!("获取数据库连接失败: {}", e))?;

    conn.execute(
        "DELETE FROM logs 
         WHERE id NOT IN (
             SELECT id FROM logs 
             ORDER BY timestamp DESC 
             LIMIT ?1
         )",
        params![keep_count],
    )
    .map_err(|e| format!("清空日志失败: {}", e))?;

    Ok(())
}

/// 导出日志
#[command]
pub fn export_logs(
    level: Option<String>,
    module: Option<String>,
    start_time: Option<String>,
    end_time: Option<String>,
) -> Result<String, String> {
    let logs = get_logs(level, module, start_time, end_time, None, 10000, 0)
        .map_err(|e| format!("获取日志失败: {}", e))?;

    // 转换为CSV格式
    let mut csv = String::from("时间,级别,模块,消息,错误详情,任务ID\n");
    
    for log in logs {
        csv.push_str(&format!(
            "{},{},{},{},{},{}\n",
            log.timestamp,
            log.level,
            log.module.unwrap_or_default(),
            escape_csv_field(&log.message),
            escape_csv_field(&log.error_detail.unwrap_or_default()),
            log.task_id.map(|id| id.to_string()).unwrap_or_default(),
        ));
    }

    Ok(csv)
}

fn escape_csv_field(field: &str) -> String {
    if field.contains(',') || field.contains('"') || field.contains('\n') {
        format!("\"{}\"", field.replace("\"", "\"\""))
    } else {
        field.to_string()
    }
}
