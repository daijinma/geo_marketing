package repositories

import "database/sql"

// SettingsRepository handles settings data.
type SettingsRepository struct {
	*BaseRepository
}

// NewSettingsRepository creates a new settings repository.
func NewSettingsRepository(db *sql.DB) *SettingsRepository {
	return &SettingsRepository{BaseRepository: NewBaseRepository(db)}
}

// Save saves a setting.
func (r *SettingsRepository) Save(key, value string) error {
	_, err := r.db.Exec(
		"INSERT OR REPLACE INTO settings (key, value, updated_at) VALUES (?, ?, datetime('now'))",
		key, value,
	)
	return err
}

// Get retrieves a setting.
func (r *SettingsRepository) Get(key string) (string, error) {
	var value string
	err := r.db.QueryRow("SELECT value FROM settings WHERE key = ?", key).Scan(&value)
	if err == sql.ErrNoRows {
		return "", nil
	}
	return value, err
}
