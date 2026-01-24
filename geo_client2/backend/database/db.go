package database

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	"github.com/google/uuid"
	_ "modernc.org/sqlite"
)

const schemaVersion = 11

var db *sql.DB

// Init initializes the database connection and runs migrations.
func Init() error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home dir: %w", err)
	}

	dbDir := filepath.Join(homeDir, ".geo_client2")
	if err := os.MkdirAll(dbDir, 0755); err != nil {
		return fmt.Errorf("failed to create db dir: %w", err)
	}

	dbPath := filepath.Join(dbDir, "cache.db")

	conn, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return fmt.Errorf("failed to open db: %w", err)
	}

	db = conn

	// Enable WAL mode and foreign keys
	if _, err := db.Exec("PRAGMA journal_mode = WAL"); err != nil {
		return fmt.Errorf("failed to set WAL mode: %w", err)
	}
	if _, err := db.Exec("PRAGMA foreign_keys = ON"); err != nil {
		return fmt.Errorf("failed to enable foreign keys: %w", err)
	}

	// Run migrations
	if err := runMigrations(); err != nil {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	// Ensure indexes that might depend on migrations
	if _, err := db.Exec("CREATE INDEX IF NOT EXISTS idx_login_status_platform_account ON login_status(platform_name, account_id)"); err != nil {
		fmt.Printf("Warning: failed to create login_status index: %v\n", err)
	}
	if _, err := db.Exec("CREATE INDEX IF NOT EXISTS idx_logs_session ON logs(session_id)"); err != nil {
		fmt.Printf("Warning: failed to create logs session index: %v\n", err)
	}
	if _, err := db.Exec("CREATE INDEX IF NOT EXISTS idx_logs_correlation ON logs(correlation_id)"); err != nil {
		fmt.Printf("Warning: failed to create logs correlation index: %v\n", err)
	}

	return nil
}

// runMigrations runs database migrations.
func runMigrations() error {
	if err := initSchema(); err != nil {
		return err
	}

	// Check current version
	var currentVersion int
	err := GetDB().QueryRow("SELECT version FROM db_version WHERE id = 1").Scan(&currentVersion)
	if err == sql.ErrNoRows {
		// First run, insert version
		_, err = GetDB().Exec("INSERT INTO db_version (id, version) VALUES (1, ?)", schemaVersion)
		if err != nil {
			return fmt.Errorf("failed to set initial version: %w", err)
		}
		return nil
	} else if err != nil {
		return fmt.Errorf("failed to get version: %w", err)
	}

	// Run migrations if needed
	if currentVersion < schemaVersion {
		// Migrate from version 6 to 7: Add accounts table and update login_status
		if currentVersion == 6 {
			if err := migrateToV7(); err != nil {
				return fmt.Errorf("failed to migrate to v7: %w", err)
			}
		}

		// Migrate from version 7 to 8: Add new log columns
		if currentVersion <= 7 {
			if err := migrateToV8(); err != nil {
				return fmt.Errorf("failed to migrate to v8: %w", err)
			}
		}

		// Migrate from version 8 to 9: Add name column to tasks
		if currentVersion <= 8 {
			if err := migrateToV9(); err != nil {
				return fmt.Errorf("failed to migrate to v9: %w", err)
			}
		}

		// Migrate from version 9 to 10: Add category column to accounts
		if currentVersion <= 9 {
			if err := migrateToV10(); err != nil {
				return fmt.Errorf("failed to migrate to v10: %w", err)
			}
		}

		// Migrate from version 10 to 11: Update category values based on platform
		if currentVersion <= 10 {
			if err := migrateToV11(); err != nil {
				return fmt.Errorf("failed to migrate to v11: %w", err)
			}
		}

		_, err = GetDB().Exec("UPDATE db_version SET version = ?, updated_at = datetime('now') WHERE id = 1", schemaVersion)
		if err != nil {
			return fmt.Errorf("failed to update version: %w", err)
		}
	}

	return nil
}

// migrateToV10 migrates database from version 9 to 10.
// Adds category column to accounts table.
func migrateToV10() error {
	db := GetDB()
	var colExists int
	db.QueryRow("SELECT COUNT(*) FROM pragma_table_info('accounts') WHERE name='category'").Scan(&colExists)
	if colExists == 0 {
		if _, err := db.Exec("ALTER TABLE accounts ADD COLUMN category TEXT DEFAULT 'ai_model'"); err != nil {
			return fmt.Errorf("failed to add category column to accounts: %w", err)
		}
		if _, err := db.Exec("CREATE INDEX IF NOT EXISTS idx_accounts_category ON accounts(category)"); err != nil {
			return fmt.Errorf("failed to create index on accounts(category): %w", err)
		}
	}
	return nil
}

// migrateToV11 migrates database from version 10 to 11.
// Updates category values based on platform type.
func migrateToV11() error {
	db := GetDB()

	// Set AI model platforms
	aiModelPlatforms := []string{"deepseek", "doubao", "yiyan", "yuanbao"}
	for _, platform := range aiModelPlatforms {
		_, err := db.Exec("UPDATE accounts SET category = 'ai_model' WHERE platform = ?", platform)
		if err != nil {
			return fmt.Errorf("failed to update category for platform %s: %w", platform, err)
		}
	}

	// Set publishing platforms
	publishingPlatforms := []string{"xiaohongshu"}
	for _, platform := range publishingPlatforms {
		_, err := db.Exec("UPDATE accounts SET category = 'publishing' WHERE platform = ?", platform)
		if err != nil {
			return fmt.Errorf("failed to update category for platform %s: %w", platform, err)
		}
	}

	return nil
}

// migrateToV9 migrates database from version 8 to 9.
// Adds name column to tasks table.
func migrateToV9() error {
	db := GetDB()
	var colExists int
	db.QueryRow("SELECT COUNT(*) FROM pragma_table_info('tasks') WHERE name='name'").Scan(&colExists)
	if colExists == 0 {
		if _, err := db.Exec("ALTER TABLE tasks ADD COLUMN name TEXT"); err != nil {
			return fmt.Errorf("failed to add name column to tasks: %w", err)
		}
	}
	return nil
}

// migrateToV7 migrates database from version 6 to 7.
// Adds accounts table and updates login_status to support multiple accounts.
func migrateToV7() error {
	db := GetDB()

	// Check if accounts table already exists
	var exists int
	err := db.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='accounts'").Scan(&exists)
	if err != nil {
		return fmt.Errorf("failed to check accounts table: %w", err)
	}

	// Create accounts table if it doesn't exist
	if exists == 0 {
		_, err = db.Exec(`
			CREATE TABLE IF NOT EXISTS accounts (
				id INTEGER PRIMARY KEY AUTOINCREMENT,
				platform TEXT NOT NULL,
				account_id TEXT NOT NULL UNIQUE,
				account_name TEXT,
				user_data_dir TEXT NOT NULL,
				is_active INTEGER DEFAULT 1,
				created_at TEXT DEFAULT (datetime('now', 'localtime')),
				updated_at TEXT DEFAULT (datetime('now', 'localtime')),
				UNIQUE(platform, account_id)
			)
		`)
		if err != nil {
			return fmt.Errorf("failed to create accounts table: %w", err)
		}

		// Create indexes
		_, err = db.Exec("CREATE INDEX IF NOT EXISTS idx_accounts_platform ON accounts(platform)")
		if err != nil {
			return fmt.Errorf("failed to create accounts index: %w", err)
		}
		_, err = db.Exec("CREATE INDEX IF NOT EXISTS idx_accounts_active ON accounts(platform, is_active)")
		if err != nil {
			return fmt.Errorf("failed to create accounts active index: %w", err)
		}
	}

	// Check if login_status has account_id column
	var columnExists int
	err = db.QueryRow(`
		SELECT COUNT(*) FROM pragma_table_info('login_status') WHERE name='account_id'
	`).Scan(&columnExists)
	if err != nil {
		return fmt.Errorf("failed to check login_status columns: %w", err)
	}

	// Add account_id column if it doesn't exist
	if columnExists == 0 {
		// First, drop the old UNIQUE constraint on platform_name
		// SQLite doesn't support DROP CONSTRAINT, so we need to recreate the table
		_, err = db.Exec(`
			CREATE TABLE IF NOT EXISTS login_status_new (
				id INTEGER PRIMARY KEY AUTOINCREMENT,
				platform_type TEXT NOT NULL,
				platform_name TEXT NOT NULL,
				account_id TEXT,
				is_logged_in INTEGER NOT NULL DEFAULT 0,
				last_check_at TEXT,
				updated_at TEXT DEFAULT (datetime('now', 'localtime')),
				UNIQUE(platform_name, account_id)
			)
		`)
		if err != nil {
			return fmt.Errorf("failed to create new login_status table: %w", err)
		}

		// Migrate existing data (set account_id to NULL for existing records)
		_, err = db.Exec(`
			INSERT INTO login_status_new (id, platform_type, platform_name, account_id, is_logged_in, last_check_at, updated_at)
			SELECT id, platform_type, platform_name, NULL, is_logged_in, last_check_at, updated_at
			FROM login_status
		`)
		if err != nil {
			return fmt.Errorf("failed to migrate login_status data: %w", err)
		}

		// Drop old table and rename new one
		_, err = db.Exec("DROP TABLE login_status")
		if err != nil {
			return fmt.Errorf("failed to drop old login_status table: %w", err)
		}
		_, err = db.Exec("ALTER TABLE login_status_new RENAME TO login_status")
		if err != nil {
			return fmt.Errorf("failed to rename login_status table: %w", err)
		}

		// Create index
		_, err = db.Exec("CREATE INDEX IF NOT EXISTS idx_login_status_platform_account ON login_status(platform_name, account_id)")
		if err != nil {
			return fmt.Errorf("failed to create login_status index: %w", err)
		}

		// Create default accounts for existing platforms
		// Scan existing browser_data directories and create accounts
		if err := createDefaultAccountsForExistingPlatforms(); err != nil {
			// Log error but don't fail migration
			fmt.Printf("Warning: failed to create default accounts: %v\n", err)
		}
	}

	return nil
}

// createDefaultAccountsForExistingPlatforms creates default accounts for existing platforms.
func createDefaultAccountsForExistingPlatforms() error {
	db := GetDB()
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	platforms := []string{"deepseek", "doubao"}

	for _, platform := range platforms {
		// Check if platform already has accounts
		var count int
		err := db.QueryRow("SELECT COUNT(*) FROM accounts WHERE platform = ?", platform).Scan(&count)
		if err != nil {
			continue
		}

		if count > 0 {
			continue // Platform already has accounts
		}

		// Check if browser_data directory exists
		browserDataDir := filepath.Join(homeDir, ".geo_client2", "browser_data", platform)
		if _, err := os.Stat(browserDataDir); os.IsNotExist(err) {
			continue // No existing browser data
		}

		// Generate UUID for account_id
		accountID := generateUUID()
		accountName := fmt.Sprintf("%s (默认账号)", platform)

		// Determine user_data_dir based on existing structure
		// If old structure exists (direct platform folder), we'll migrate it
		userDataDir := filepath.Join(browserDataDir, accountID)

		// Create account
		_, err = db.Exec(`
			INSERT INTO accounts (platform, account_id, account_name, user_data_dir, is_active)
			VALUES (?, ?, ?, ?, 1)
		`, platform, accountID, accountName, userDataDir)
		if err != nil {
			continue
		}

		// Migrate existing login_status to this account
		_, err = db.Exec(`
			UPDATE login_status SET account_id = ? WHERE platform_name = ? AND account_id IS NULL
		`, accountID, platform)
		if err != nil {
			// Continue even if update fails
			continue
		}

		// If old browser_data/{platform} exists, rename it to browser_data/{platform}/{account_id}
		if _, err := os.Stat(browserDataDir); err == nil {
			// Check if it's a directory (not already migrated)
			if info, err := os.Stat(browserDataDir); err == nil && info.IsDir() {
				// Check if it's the old structure (no account_id subdirectories)
				files, _ := os.ReadDir(browserDataDir)
				hasAccountDirs := false
				for _, file := range files {
					if file.IsDir() && len(file.Name()) == 36 { // UUID length
						hasAccountDirs = true
						break
					}
				}

				if !hasAccountDirs {
					// Migrate: rename old directory to new structure
					tempDir := browserDataDir + "_old"
					if err := os.Rename(browserDataDir, tempDir); err == nil {
						os.MkdirAll(browserDataDir, 0755)
						if err := os.Rename(tempDir, userDataDir); err != nil {
							// Rollback
							os.Rename(tempDir, browserDataDir)
						}
					}
				}
			}
		}
	}

	return nil
}

// migrateToV8 migrates database from version 7 to 8.
// Adds new columns to logs table for structured logging.
func migrateToV8() error {
	db := GetDB()

	// Check if columns already exist
	var sessionColExists, correlationColExists, componentColExists, userActionColExists, perfColExists int
	db.QueryRow("SELECT COUNT(*) FROM pragma_table_info('logs') WHERE name='session_id'").Scan(&sessionColExists)
	db.QueryRow("SELECT COUNT(*) FROM pragma_table_info('logs') WHERE name='correlation_id'").Scan(&correlationColExists)
	db.QueryRow("SELECT COUNT(*) FROM pragma_table_info('logs') WHERE name='component'").Scan(&componentColExists)
	db.QueryRow("SELECT COUNT(*) FROM pragma_table_info('logs') WHERE name='user_action'").Scan(&userActionColExists)
	db.QueryRow("SELECT COUNT(*) FROM pragma_table_info('logs') WHERE name='performance_ms'").Scan(&perfColExists)

	// Add columns if they don't exist
	if sessionColExists == 0 {
		if _, err := db.Exec("ALTER TABLE logs ADD COLUMN session_id TEXT"); err != nil {
			return fmt.Errorf("failed to add session_id column: %w", err)
		}
	}
	if correlationColExists == 0 {
		if _, err := db.Exec("ALTER TABLE logs ADD COLUMN correlation_id TEXT"); err != nil {
			return fmt.Errorf("failed to add correlation_id column: %w", err)
		}
	}
	if componentColExists == 0 {
		if _, err := db.Exec("ALTER TABLE logs ADD COLUMN component TEXT"); err != nil {
			return fmt.Errorf("failed to add component column: %w", err)
		}
	}
	if userActionColExists == 0 {
		if _, err := db.Exec("ALTER TABLE logs ADD COLUMN user_action TEXT"); err != nil {
			return fmt.Errorf("failed to add user_action column: %w", err)
		}
	}
	if perfColExists == 0 {
		if _, err := db.Exec("ALTER TABLE logs ADD COLUMN performance_ms INTEGER"); err != nil {
			return fmt.Errorf("failed to add performance_ms column: %w", err)
		}
	}

	// Create indexes for new columns
	db.Exec("CREATE INDEX IF NOT EXISTS idx_logs_session ON logs(session_id)")
	db.Exec("CREATE INDEX IF NOT EXISTS idx_logs_correlation ON logs(correlation_id)")

	return nil
}

// generateUUID generates a UUID v4.
func generateUUID() string {
	return uuid.New().String()
}

// GetDB returns the database connection.
func GetDB() *sql.DB {
	if db == nil {
		panic("database not initialized, call Init() first")
	}
	return db
}

// Close closes the database connection.
func Close() error {
	if db != nil {
		return db.Close()
	}
	return nil
}
