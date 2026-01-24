package repositories

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	"geo_client2/backend/logger"
	"github.com/google/uuid"
)

// Account represents an account entity.
type Account struct {
	ID          int    `json:"id"`
	Platform    string `json:"platform"`
	AccountID   string `json:"account_id"`
	AccountName string `json:"account_name"`
	UserDataDir string `json:"user_data_dir"`
	IsActive    bool   `json:"is_active"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
}

// AccountRepository handles account data.
type AccountRepository struct {
	*BaseRepository
}

// NewAccountRepository creates a new account repository.
func NewAccountRepository(db *sql.DB) *AccountRepository {
	return &AccountRepository{BaseRepository: NewBaseRepository(db)}
}

// CreateAccount creates a new account.
func (r *AccountRepository) CreateAccount(platform, accountName string) (*Account, error) {
	accountID := uuid.New().String()

	// Generate user_data_dir path
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home dir: %w", err)
	}
	userDataDir := filepath.Join(homeDir, ".geo_client2", "browser_data", platform, accountID)

	// Create directory
	if err := os.MkdirAll(userDataDir, 0755); err != nil {
		logger.GetLogger().Error(fmt.Sprintf("Failed to create user data dir: %s", userDataDir), err)
		return nil, fmt.Errorf("failed to create user data dir: %w", err)
	}

	// If this is the first account for the platform, set it as active
	var isActive int = 0
	var existingCount int
	err = r.db.QueryRow("SELECT COUNT(*) FROM accounts WHERE platform = ?", platform).Scan(&existingCount)
	if err == nil && existingCount == 0 {
		isActive = 1
	}

	// Insert account
	result, err := r.db.Exec(`
		INSERT INTO accounts (platform, account_id, account_name, user_data_dir, is_active)
		VALUES (?, ?, ?, ?, ?)
	`, platform, accountID, accountName, userDataDir, isActive)
	if err != nil {
		// Cleanup directory on error
		os.RemoveAll(userDataDir)
		logger.GetLogger().Error("Failed to insert account into database", err)
		return nil, fmt.Errorf("failed to create account: %w", err)
	}

	id, _ := result.LastInsertId()

	return &Account{
		ID:          int(id),
		Platform:    platform,
		AccountID:   accountID,
		AccountName: accountName,
		UserDataDir: userDataDir,
		IsActive:    isActive == 1,
	}, nil
}

// GetAccountByID retrieves an account by account_id.
func (r *AccountRepository) GetAccountByID(accountID string) (*Account, error) {
	var account Account
	var isActiveInt int
	err := r.db.QueryRow(`
		SELECT id, platform, account_id, account_name, user_data_dir, is_active, created_at, updated_at
		FROM accounts WHERE account_id = ?
	`, accountID).Scan(
		&account.ID, &account.Platform, &account.AccountID, &account.AccountName,
		&account.UserDataDir, &isActiveInt, &account.CreatedAt, &account.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get account: %w", err)
	}
	account.IsActive = isActiveInt == 1
	return &account, nil
}

// GetAccountsByPlatform retrieves all accounts for a platform.
func (r *AccountRepository) GetAccountsByPlatform(platform string) ([]Account, error) {
	rows, err := r.db.Query(`
		SELECT id, platform, account_id, account_name, user_data_dir, is_active, created_at, updated_at
		FROM accounts WHERE platform = ? ORDER BY is_active DESC, created_at ASC
	`, platform)
	if err != nil {
		return nil, fmt.Errorf("failed to get accounts: %w", err)
	}
	defer rows.Close()

	var accounts []Account
	for rows.Next() {
		var account Account
		var isActiveInt int
		if err := rows.Scan(
			&account.ID, &account.Platform, &account.AccountID, &account.AccountName,
			&account.UserDataDir, &isActiveInt, &account.CreatedAt, &account.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan account: %w", err)
		}
		account.IsActive = isActiveInt == 1
		accounts = append(accounts, account)
	}
	return accounts, nil
}

// GetActiveAccount retrieves the active account for a platform.
func (r *AccountRepository) GetActiveAccount(platform string) (*Account, error) {
	var account Account
	var isActiveInt int
	err := r.db.QueryRow(`
		SELECT id, platform, account_id, account_name, user_data_dir, is_active, created_at, updated_at
		FROM accounts WHERE platform = ? AND is_active = 1 LIMIT 1
	`, platform).Scan(
		&account.ID, &account.Platform, &account.AccountID, &account.AccountName,
		&account.UserDataDir, &isActiveInt, &account.CreatedAt, &account.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get active account: %w", err)
	}
	account.IsActive = isActiveInt == 1
	return &account, nil
}

// SetActiveAccount sets an account as active for its platform.
func (r *AccountRepository) SetActiveAccount(platform, accountID string) error {
	// Start transaction
	tx, err := r.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Set all accounts for this platform to inactive
	_, err = tx.Exec("UPDATE accounts SET is_active = 0 WHERE platform = ?", platform)
	if err != nil {
		return fmt.Errorf("failed to deactivate accounts: %w", err)
	}

	// Set the specified account as active
	_, err = tx.Exec("UPDATE accounts SET is_active = 1, updated_at = datetime('now') WHERE account_id = ?", accountID)
	if err != nil {
		return fmt.Errorf("failed to activate account: %w", err)
	}

	return tx.Commit()
}

// DeleteAccount deletes an account and its user data directory.
func (r *AccountRepository) DeleteAccount(accountID string) error {
	// Get account info first
	account, err := r.GetAccountByID(accountID)
	if err != nil {
		return fmt.Errorf("failed to get account: %w", err)
	}
	if account == nil {
		return fmt.Errorf("account not found")
	}

	// Start transaction
	tx, err := r.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Delete account
	_, err = tx.Exec("DELETE FROM accounts WHERE account_id = ?", accountID)
	if err != nil {
		return fmt.Errorf("failed to delete account: %w", err)
	}

	// If this was the active account, set another account as active (if exists)
	if account.IsActive {
		var newActiveID string
		err = tx.QueryRow(`
			SELECT account_id FROM accounts 
			WHERE platform = ? AND account_id != ? 
			ORDER BY created_at ASC LIMIT 1
		`, account.Platform, accountID).Scan(&newActiveID)
		if err == nil {
			_, err = tx.Exec("UPDATE accounts SET is_active = 1 WHERE account_id = ?", newActiveID)
			if err != nil {
				return fmt.Errorf("failed to set new active account: %w", err)
			}
		}
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	// Delete user data directory
	if err := os.RemoveAll(account.UserDataDir); err != nil {
		// Log error but don't fail (directory might not exist)
		fmt.Printf("Warning: failed to delete user data dir %s: %v\n", account.UserDataDir, err)
	}

	return nil
}

// UpdateAccountName updates an account's display name.
func (r *AccountRepository) UpdateAccountName(accountID, name string) error {
	_, err := r.db.Exec(`
		UPDATE accounts SET account_name = ?, updated_at = datetime('now') WHERE account_id = ?
	`, name, accountID)
	if err != nil {
		return fmt.Errorf("failed to update account name: %w", err)
	}
	return nil
}
