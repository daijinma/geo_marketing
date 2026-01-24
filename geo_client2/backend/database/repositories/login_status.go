package repositories

import (
	"database/sql"
)

// LoginStatusRepository handles login status data.
type LoginStatusRepository struct {
	*BaseRepository
}

// NewLoginStatusRepository creates a new login status repository.
func NewLoginStatusRepository(db *sql.DB) *LoginStatusRepository {
	return &LoginStatusRepository{BaseRepository: NewBaseRepository(db)}
}

// Save saves login status with account_id support.
func (r *LoginStatusRepository) Save(platformType, platformName string, isLoggedIn bool, lastCheckAt *string, accountID *string) error {
	isLoggedInInt := 0
	if isLoggedIn {
		isLoggedInInt = 1
	}
	_, err := r.db.Exec(
		"INSERT OR REPLACE INTO login_status (platform_type, platform_name, account_id, is_logged_in, last_check_at, updated_at) VALUES (?, ?, ?, ?, ?, datetime('now'))",
		platformType, platformName, accountID, isLoggedInInt, lastCheckAt,
	)
	return err
}

// GetByPlatform retrieves login status for a platform (backward compatibility - uses first account if multiple).
func (r *LoginStatusRepository) GetByPlatform(platformName string) (platformType string, isLoggedIn bool, err error) {
	var isLoggedInInt int
	var accountID sql.NullString
	err = r.db.QueryRow("SELECT platform_type, account_id, is_logged_in FROM login_status WHERE platform_name = ? LIMIT 1", platformName).
		Scan(&platformType, &accountID, &isLoggedInInt)
	if err == sql.ErrNoRows {
		return "", false, nil
	}
	isLoggedIn = isLoggedInInt == 1
	return platformType, isLoggedIn, err
}

// GetByPlatformAndAccount retrieves login status for a platform and account.
func (r *LoginStatusRepository) GetByPlatformAndAccount(platformName string, accountID string) (platformType string, isLoggedIn bool, err error) {
	var isLoggedInInt int
	err = r.db.QueryRow("SELECT platform_type, is_logged_in FROM login_status WHERE platform_name = ? AND account_id = ?", platformName, accountID).
		Scan(&platformType, &isLoggedInInt)
	if err == sql.ErrNoRows {
		return "", false, nil
	}
	isLoggedIn = isLoggedInInt == 1
	return platformType, isLoggedIn, err
}

// UpdateLoginStatus updates the login status for an account.
func (r *LoginStatusRepository) UpdateLoginStatus(platformName, accountID string, isLoggedIn bool) error {
	isLoggedInInt := 0
	if isLoggedIn {
		isLoggedInInt = 1
	}
	// Use INSERT OR IGNORE to ensure row exists, then UPDATE
	// Or simpler: just use Save if we know platformType.
	// Let's assume platformType = platformName for now if new, or fetch existing.

	var platformType string
	err := r.db.QueryRow("SELECT platform_type FROM login_status WHERE platform_name = ? AND account_id = ?", platformName, accountID).Scan(&platformType)
	if err == sql.ErrNoRows {
		platformType = "search_engine" // Default
	}

	_, err = r.db.Exec(`
		INSERT INTO login_status (platform_type, platform_name, account_id, is_logged_in, last_check_at, updated_at)
		VALUES (?, ?, ?, ?, datetime('now'), datetime('now'))
		ON CONFLICT(platform_name, account_id) DO UPDATE SET
		is_logged_in = excluded.is_logged_in,
		last_check_at = excluded.last_check_at,
		updated_at = excluded.updated_at
	`, platformType, platformName, accountID, isLoggedInInt)

	return err
}

// GetAll retrieves all login statuses.
func (r *LoginStatusRepository) GetAll() ([]map[string]interface{}, error) {
	rows, err := r.db.Query("SELECT * FROM login_status ORDER BY platform_name")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []map[string]interface{}
	cols, _ := rows.Columns()
	for rows.Next() {
		values := make([]interface{}, len(cols))
		valuePtrs := make([]interface{}, len(cols))
		for i := range values {
			valuePtrs[i] = &values[i]
		}
		if err := rows.Scan(valuePtrs...); err != nil {
			return nil, err
		}
		entry := make(map[string]interface{})
		for i, col := range cols {
			entry[col] = values[i]
		}
		results = append(results, entry)
	}
	return results, nil
}
