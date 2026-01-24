package database

import (
	"fmt"
)

// initSchema creates all tables if they don't exist.
func initSchema() error {
	db := GetDB()

	schemas := []string{
		`CREATE TABLE IF NOT EXISTS tasks (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			task_id INTEGER UNIQUE,
			name TEXT,
			keywords TEXT NOT NULL,
			platforms TEXT NOT NULL,
			query_count INTEGER NOT NULL DEFAULT 1,
			status TEXT NOT NULL DEFAULT 'pending',
			result_data TEXT,
			task_type TEXT DEFAULT 'local_search',
			source TEXT DEFAULT 'local',
			server_task_id INTEGER,
			created_by TEXT,
			account_id TEXT,
			priority INTEGER DEFAULT 0,
			scheduled_at TEXT,
			total_queries INTEGER DEFAULT 0,
			completed_queries INTEGER DEFAULT 0,
			total_records INTEGER DEFAULT 0,
			completed_records INTEGER DEFAULT 0,
			failed_records INTEGER DEFAULT 0,
			total_citations INTEGER DEFAULT 0,
			settings TEXT,
			created_at TEXT DEFAULT (datetime('now', 'localtime')),
			updated_at TEXT DEFAULT (datetime('now', 'localtime'))
		)`,
		`CREATE TABLE IF NOT EXISTS accounts (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			platform TEXT NOT NULL,
			account_id TEXT NOT NULL UNIQUE,
			account_name TEXT NOT NULL,
			user_data_dir TEXT NOT NULL,
			is_active INTEGER DEFAULT 0,
			category TEXT DEFAULT 'ai_model',
			created_at TEXT DEFAULT (datetime('now', 'localtime')),
			updated_at TEXT DEFAULT (datetime('now', 'localtime')),
			UNIQUE(platform, account_id)
		)`,
		`CREATE TABLE IF NOT EXISTS login_status (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			platform_type TEXT NOT NULL,
			platform_name TEXT NOT NULL,
			account_id TEXT,
			is_logged_in INTEGER NOT NULL DEFAULT 0,
			last_check_at TEXT,
			updated_at TEXT DEFAULT (datetime('now', 'localtime')),
			UNIQUE(platform_name, account_id)
		)`,
		`CREATE TABLE IF NOT EXISTS settings (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			key TEXT NOT NULL UNIQUE,
			value TEXT NOT NULL,
			updated_at TEXT DEFAULT (datetime('now', 'localtime'))
		)`,
		`CREATE TABLE IF NOT EXISTS logs (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			level TEXT NOT NULL,
			source TEXT NOT NULL,
			message TEXT NOT NULL,
			details TEXT,
			task_id INTEGER,
			session_id TEXT,
			correlation_id TEXT,
			component TEXT,
			user_action TEXT,
			performance_ms INTEGER,
			timestamp TEXT DEFAULT (datetime('now', 'localtime'))
		)`,
		`CREATE TABLE IF NOT EXISTS search_records (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			task_id INTEGER,
			keyword TEXT NOT NULL,
			platform TEXT NOT NULL,
			prompt_type TEXT DEFAULT 'local_task',
			prompt TEXT,
			full_answer TEXT,
			response_time_ms INTEGER,
			search_status TEXT DEFAULT 'completed',
			error_message TEXT,
			round_number INTEGER DEFAULT 1,
			created_at TEXT DEFAULT (datetime('now', 'localtime')),
			FOREIGN KEY (task_id) REFERENCES tasks(id) ON DELETE CASCADE
		)`,
		`CREATE TABLE IF NOT EXISTS citations (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			record_id INTEGER NOT NULL,
			cite_index INTEGER DEFAULT 0,
			url TEXT NOT NULL,
			domain TEXT,
			title TEXT,
			snippet TEXT,
			site_name TEXT,
			query_indexes TEXT,
			created_at TEXT DEFAULT (datetime('now', 'localtime')),
			FOREIGN KEY (record_id) REFERENCES search_records(id) ON DELETE CASCADE,
			UNIQUE(record_id, url)
		)`,
		`CREATE TABLE IF NOT EXISTS search_queries (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			record_id INTEGER NOT NULL,
			query TEXT NOT NULL,
			query_order INTEGER DEFAULT 0,
			created_at TEXT DEFAULT (datetime('now', 'localtime')),
			FOREIGN KEY (record_id) REFERENCES search_records(id) ON DELETE CASCADE
		)`,
		`CREATE TABLE IF NOT EXISTS domain_stats (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			domain TEXT NOT NULL UNIQUE,
			deepseek_count INTEGER DEFAULT 0,
			doubao_count INTEGER DEFAULT 0,
			bocha_count INTEGER DEFAULT 0,
			total_count INTEGER DEFAULT 0,
			first_seen_at TEXT DEFAULT (datetime('now', 'localtime')),
			last_seen_at TEXT DEFAULT (datetime('now', 'localtime'))
		)`,
		`CREATE TABLE IF NOT EXISTS db_version (
			id INTEGER PRIMARY KEY CHECK (id = 1),
			version INTEGER NOT NULL DEFAULT 1,
			updated_at TEXT DEFAULT (datetime('now', 'localtime'))
		)`,
	}

	indexes := []string{
		"CREATE INDEX IF NOT EXISTS idx_tasks_status ON tasks(status)",
		"CREATE INDEX IF NOT EXISTS idx_tasks_type ON tasks(task_type)",
		"CREATE INDEX IF NOT EXISTS idx_tasks_source ON tasks(source)",
		"CREATE INDEX IF NOT EXISTS idx_tasks_created_by ON tasks(created_by)",
		"CREATE INDEX IF NOT EXISTS idx_sr_task ON search_records(task_id)",
		"CREATE INDEX IF NOT EXISTS idx_sr_platform ON search_records(platform)",
		"CREATE INDEX IF NOT EXISTS idx_sr_status ON search_records(search_status)",
		"CREATE INDEX IF NOT EXISTS idx_sr_round ON search_records(round_number)",
		"CREATE INDEX IF NOT EXISTS idx_citations_record ON citations(record_id)",
		"CREATE INDEX IF NOT EXISTS idx_citations_domain ON citations(domain)",
		"CREATE INDEX IF NOT EXISTS idx_sq_record ON search_queries(record_id)",
		"CREATE INDEX IF NOT EXISTS idx_ds_domain ON domain_stats(domain)",
		"CREATE INDEX IF NOT EXISTS idx_accounts_platform ON accounts(platform)",
		"CREATE INDEX IF NOT EXISTS idx_accounts_active ON accounts(platform, is_active)",
		// "CREATE INDEX IF NOT EXISTS idx_accounts_category ON accounts(category)", // Handled in db.go/migration to avoid failures on older schemas
		"CREATE INDEX IF NOT EXISTS idx_logs_level ON logs(level)",
		"CREATE INDEX IF NOT EXISTS idx_logs_source ON logs(source)",
		// "CREATE INDEX IF NOT EXISTS idx_logs_session ON logs(session_id)", // Handled in migration/db.go
		// "CREATE INDEX IF NOT EXISTS idx_logs_correlation ON logs(correlation_id)", // Handled in migration/db.go
		"CREATE INDEX IF NOT EXISTS idx_logs_task ON logs(task_id)",
		"CREATE INDEX IF NOT EXISTS idx_logs_timestamp ON logs(timestamp)",
		// "CREATE INDEX IF NOT EXISTS idx_login_status_platform_account ON login_status(platform_name, account_id)", // Created manually in db.go to handle migrations
	}

	for _, schema := range schemas {
		if _, err := db.Exec(schema); err != nil {
			return fmt.Errorf("failed to create table: %w", err)
		}
	}

	for _, idx := range indexes {
		if _, err := db.Exec(idx); err != nil {
			return fmt.Errorf("failed to create index: %w", err)
		}
	}

	return nil
}
