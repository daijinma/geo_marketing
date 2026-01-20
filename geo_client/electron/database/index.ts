import Database from 'better-sqlite3';
import { homedir } from 'os';
import { join } from 'path';
import { existsSync, mkdirSync } from 'fs';
import { parseISO, isAfter } from 'date-fns';

const DB_DIR = join(homedir(), '.geo_client');
const DB_PATH = join(DB_DIR, 'cache.db');

let db: Database.Database | null = null;

/**
 * 获取数据库连接
 */
function getDatabase(): Database.Database {
  if (db) {
    return db;
  }

  // 确保目录存在
  if (!existsSync(DB_DIR)) {
    mkdirSync(DB_DIR, { recursive: true });
  }

  db = new Database(DB_PATH);
  
  // 启用外键约束
  db.pragma('foreign_keys = ON');
  
  // 初始化表
  initTables(db);
  
  // 更新 schema 版本
  updateSchemaVersion(db);

  return db;
}

/**
 * 初始化所有表
 */
function initTables(database: Database.Database): void {
  // auth 表：存储认证token
  database.exec(`
    CREATE TABLE IF NOT EXISTS auth (
      id INTEGER PRIMARY KEY AUTOINCREMENT,
      token TEXT NOT NULL,
      expires_at TEXT NOT NULL,
      created_at TEXT DEFAULT CURRENT_TIMESTAMP,
      updated_at TEXT DEFAULT CURRENT_TIMESTAMP
    )
  `);

  // tasks 表：本地任务数据
  database.exec(`
    CREATE TABLE IF NOT EXISTS tasks (
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
    )
  `);

  // 添加索引
  database.exec(`
    CREATE INDEX IF NOT EXISTS idx_tasks_status ON tasks(status)
  `);

  // login_status 表：平台登录状态
  database.exec(`
    CREATE TABLE IF NOT EXISTS login_status (
      id INTEGER PRIMARY KEY AUTOINCREMENT,
      platform_type TEXT NOT NULL,
      platform_name TEXT NOT NULL,
      is_logged_in INTEGER NOT NULL DEFAULT 0,
      last_check_at TEXT,
      updated_at TEXT DEFAULT CURRENT_TIMESTAMP,
      UNIQUE(platform_name)
    )
  `);

  // 添加索引
  database.exec(`
    CREATE INDEX IF NOT EXISTS idx_login_status_platform ON login_status(platform_name)
  `);

  // settings 表：用户设置
  database.exec(`
    CREATE TABLE IF NOT EXISTS settings (
      key TEXT PRIMARY KEY,
      value TEXT NOT NULL
    )
  `);

  // logs 表：日志记录
  database.exec(`
    CREATE TABLE IF NOT EXISTS logs (
      id INTEGER PRIMARY KEY AUTOINCREMENT,
      timestamp TEXT NOT NULL,
      level TEXT NOT NULL,
      module TEXT,
      message TEXT NOT NULL,
      error_detail TEXT,
      task_id INTEGER,
      created_at TEXT DEFAULT CURRENT_TIMESTAMP
    )
  `);

  // 添加索引
  database.exec(`
    CREATE INDEX IF NOT EXISTS idx_logs_timestamp ON logs(timestamp)
  `);
  database.exec(`
    CREATE INDEX IF NOT EXISTS idx_logs_level ON logs(level)
  `);
  database.exec(`
    CREATE INDEX IF NOT EXISTS idx_logs_module ON logs(module)
  `);
  database.exec(`
    CREATE INDEX IF NOT EXISTS idx_logs_task_id ON logs(task_id)
  `);
}

/**
 * 更新数据库 schema 版本
 */
function updateSchemaVersion(database: Database.Database): void {
  // 创建 schema_version 表（如果不存在）
  database.exec(`
    CREATE TABLE IF NOT EXISTS schema_version (
      version INTEGER PRIMARY KEY
    )
  `);

  // 检查当前版本
  const stmt = database.prepare('SELECT version FROM schema_version LIMIT 1');
  const result = stmt.get() as { version: number } | undefined;
  const currentVersion = result?.version || 0;

  const LATEST_VERSION = 1;

  // 如果需要更新，执行迁移
  if (currentVersion < LATEST_VERSION) {
    // 迁移逻辑（当前版本1，暂时不需要迁移）
    const updateStmt = database.prepare('INSERT OR REPLACE INTO schema_version (version) VALUES (?)');
    updateStmt.run(LATEST_VERSION);
  }
}

/**
 * 初始化数据库
 */
export function initDb(): void {
  getDatabase();
}

/**
 * 保存认证token
 */
export function saveAuthToken(token: string, expiresAt: string): void {
  const database = getDatabase();
  
  // 先删除旧的token（只保留一个）
  database.prepare('DELETE FROM auth').run();
  
  // 插入新token
  const stmt = database.prepare(`
    INSERT INTO auth (token, expires_at, updated_at) 
    VALUES (?, ?, datetime('now'))
  `);
  stmt.run(token, expiresAt);
}

/**
 * 获取认证token
 */
export function getAuthToken(): { token: string; expires_at: string } | null {
  const database = getDatabase();
  
  const stmt = database.prepare(`
    SELECT token, expires_at FROM auth ORDER BY id DESC LIMIT 1
  `);
  const result = stmt.get() as { token: string; expires_at: string } | undefined;
  
  return result || null;
}

/**
 * 删除认证token
 */
export function deleteAuthToken(): void {
  const database = getDatabase();
  database.prepare('DELETE FROM auth').run();
}

/**
 * 检查token是否过期
 */
export function isTokenExpired(): boolean {
  const tokenInfo = getAuthToken();
  
  if (!tokenInfo) {
    return true; // 没有token视为过期
  }

  try {
    // 尝试解析各种日期格式
    let expiresAt: Date;
    
    try {
      // 尝试 RFC3339 格式
      expiresAt = parseISO(tokenInfo.expires_at);
    } catch {
      // 尝试其他格式
      expiresAt = new Date(tokenInfo.expires_at);
    }

    // 检查是否过期
    return isAfter(new Date(), expiresAt);
  } catch (error) {
    console.error('解析过期时间失败:', error);
    return true; // 解析失败视为过期
  }
}

/**
 * 保存任务
 */
export function saveTask(
  taskId: number | null,
  keywords: string,
  platforms: string,
  queryCount: number,
  status: string,
  resultData: string | null = null
): void {
  const database = getDatabase();
  
  const stmt = database.prepare(`
    INSERT OR REPLACE INTO tasks 
    (task_id, keywords, platforms, query_count, status, result_data, updated_at)
    VALUES (?, ?, ?, ?, ?, ?, datetime('now'))
  `);
  stmt.run(taskId, keywords, platforms, queryCount, status, resultData);
}

/**
 * 获取任务
 */
export function getTask(taskId: number): {
  id: number;
  task_id: number | null;
  keywords: string;
  platforms: string;
  query_count: number;
  status: string;
  result_data: string | null;
  created_at: string;
  updated_at: string;
} | null {
  const database = getDatabase();
  
  const stmt = database.prepare('SELECT * FROM tasks WHERE task_id = ?');
  const result = stmt.get(taskId) as any;
  
  return result || null;
}

/**
 * 更新任务状态
 */
export function updateTaskStatus(taskId: number, status: string, resultData: string | null = null): void {
  const database = getDatabase();
  
  const stmt = database.prepare(`
    UPDATE tasks 
    SET status = ?, result_data = ?, updated_at = datetime('now')
    WHERE task_id = ?
  `);
  stmt.run(status, resultData, taskId);
}

/**
 * 保存登录状态
 */
export function saveLoginStatus(
  platformType: string,
  platformName: string,
  isLoggedIn: boolean,
  lastCheckAt: string | null = null
): void {
  const database = getDatabase();
  
  const stmt = database.prepare(`
    INSERT OR REPLACE INTO login_status 
    (platform_type, platform_name, is_logged_in, last_check_at, updated_at)
    VALUES (?, ?, ?, ?, datetime('now'))
  `);
  stmt.run(platformType, platformName, isLoggedIn ? 1 : 0, lastCheckAt);
}

/**
 * 获取登录状态
 */
export function getLoginStatus(platformName: string): {
  id: number;
  platform_type: string;
  platform_name: string;
  is_logged_in: number;
  last_check_at: string | null;
  updated_at: string;
} | null {
  const database = getDatabase();
  
  const stmt = database.prepare('SELECT * FROM login_status WHERE platform_name = ?');
  const result = stmt.get(platformName) as any;
  
  return result || null;
}

/**
 * 获取所有登录状态
 */
export function getAllLoginStatus(): Array<{
  id: number;
  platform_type: string;
  platform_name: string;
  is_logged_in: number;
  last_check_at: string | null;
  updated_at: string;
}> {
  const database = getDatabase();
  
  const stmt = database.prepare('SELECT * FROM login_status ORDER BY platform_name');
  return stmt.all() as any[];
}

/**
 * 保存设置
 */
export function saveSetting(key: string, value: string): void {
  const database = getDatabase();
  
  const stmt = database.prepare('INSERT OR REPLACE INTO settings (key, value) VALUES (?, ?)');
  stmt.run(key, value);
}

/**
 * 获取设置
 */
export function getSetting(key: string): string | null {
  const database = getDatabase();
  
  const stmt = database.prepare('SELECT value FROM settings WHERE key = ?');
  const result = stmt.get(key) as { value: string } | undefined;
  
  return result?.value || null;
}

/**
 * 添加日志
 */
export function addLog(
  timestamp: string,
  level: string,
  module: string | null,
  message: string,
  errorDetail: string | null = null,
  taskId: number | null = null
): void {
  const database = getDatabase();
  
  const stmt = database.prepare(`
    INSERT INTO logs 
    (timestamp, level, module, message, error_detail, task_id, created_at)
    VALUES (?, ?, ?, ?, ?, ?, datetime('now'))
  `);
  stmt.run(timestamp, level, module, message, errorDetail, taskId);
}

/**
 * 获取日志
 */
export function getLogs(
  limit: number = 100,
  level: string | null = null,
  module: string | null = null,
  taskId: number | null = null
): Array<{
  id: number;
  timestamp: string;
  level: string;
  module: string | null;
  message: string;
  error_detail: string | null;
  task_id: number | null;
  created_at: string;
}> {
  const database = getDatabase();
  
  let query = 'SELECT * FROM logs WHERE 1=1';
  const params: any[] = [];
  
  if (level) {
    query += ' AND level = ?';
    params.push(level);
  }
  
  if (module) {
    query += ' AND module = ?';
    params.push(module);
  }
  
  if (taskId !== null) {
    query += ' AND task_id = ?';
    params.push(taskId);
  }
  
  query += ' ORDER BY created_at DESC LIMIT ?';
  params.push(limit);
  
  const stmt = database.prepare(query);
  return stmt.all(...params) as any[];
}

/**
 * 关闭数据库连接
 */
export function closeDb(): void {
  if (db) {
    db.close();
    db = null;
  }
}
