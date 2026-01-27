package repositories

import (
	"database/sql"
	"time"
)

// AuthRepository handles authentication data.
type AuthRepository struct {
	*BaseRepository
}

// NewAuthRepository creates a new auth repository.
func NewAuthRepository(db *sql.DB) *AuthRepository {
	return &AuthRepository{BaseRepository: NewBaseRepository(db)}
}

// SaveToken saves an authentication token.
func (r *AuthRepository) SaveToken(token, expiresAt string) error {
	// Delete old tokens
	if _, err := r.db.Exec("DELETE FROM auth"); err != nil {
		return err
	}

	// Insert new token
	_, err := r.db.Exec(
		"INSERT INTO auth (token, expires_at, updated_at) VALUES (?, ?, datetime('now'))",
		token, expiresAt,
	)
	return err
}

// GetToken retrieves the authentication token.
func (r *AuthRepository) GetToken() (token, expiresAt string, err error) {
	err = r.db.QueryRow("SELECT token, expires_at FROM auth ORDER BY id DESC LIMIT 1").
		Scan(&token, &expiresAt)
	if err == sql.ErrNoRows {
		return "", "", nil
	}
	return token, expiresAt, err
}

// DeleteToken deletes the authentication token.
func (r *AuthRepository) DeleteToken() error {
	_, err := r.db.Exec("DELETE FROM auth")
	return err
}

// IsTokenExpired checks if the token is expired.
func (r *AuthRepository) IsTokenExpired() (bool, error) {
	token, expiresAt, err := r.GetToken()
	if err != nil {
		return true, err
	}
	if token == "" {
		return true, nil
	}

	expires, err := time.Parse(time.RFC3339, expiresAt)
	if err != nil {
		// Try other formats
		expires, err = time.Parse("2006-01-02 15:04:05", expiresAt)
		if err != nil {
			return true, err
		}
	}

	return time.Now().After(expires), nil
}
