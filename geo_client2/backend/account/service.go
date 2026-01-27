package account

import (
	"fmt"
	"geo_client2/backend/database/repositories"
	"geo_client2/backend/logger"
)

// Service handles account management.
type Service struct {
	accountRepo *repositories.AccountRepository
}

// NewService creates a new account service.
func NewService(accountRepo *repositories.AccountRepository) *Service {
	return &Service{accountRepo: accountRepo}
}

// CreateAccount creates a new account for a platform.
func (s *Service) CreateAccount(platform, accountName string) (*repositories.Account, error) {
	l := logger.GetLogger()
	l.Debug(fmt.Sprintf("Service.CreateAccount called for %s", platform))
	if platform == "" {
		return nil, fmt.Errorf("platform is required")
	}
	if accountName == "" {
		accountName = fmt.Sprintf("%s 账号", platform)
	}
	acc, err := s.accountRepo.CreateAccount(platform, accountName)
	if err != nil {
		l.Error("Service failed to create account", err)
		return nil, err
	}
	return acc, nil
}

// ListAccounts retrieves all accounts for a platform.
func (s *Service) ListAccounts(platform string) ([]repositories.Account, error) {
	return s.accountRepo.GetAccountsByPlatform(platform)
}

// GetActiveAccount retrieves the active account for a platform.
func (s *Service) GetActiveAccount(platform string) (*repositories.Account, error) {
	return s.accountRepo.GetActiveAccount(platform)
}

// SetActiveAccount sets an account as active for its platform.
func (s *Service) SetActiveAccount(platform, accountID string) error {
	if platform == "" || accountID == "" {
		return fmt.Errorf("platform and accountID are required")
	}
	return s.accountRepo.SetActiveAccount(platform, accountID)
}

// DeleteAccount deletes an account and its user data directory.
func (s *Service) DeleteAccount(accountID string) error {
	if accountID == "" {
		return fmt.Errorf("accountID is required")
	}
	return s.accountRepo.DeleteAccount(accountID)
}

// UpdateAccountName updates an account's display name.
func (s *Service) UpdateAccountName(accountID, name string) error {
	if accountID == "" || name == "" {
		return fmt.Errorf("accountID and name are required")
	}
	return s.accountRepo.UpdateAccountName(accountID, name)
}

// GetAccountByID retrieves an account by its ID.
func (s *Service) GetAccountByID(accountID string) (*repositories.Account, error) {
	if accountID == "" {
		return nil, fmt.Errorf("accountID is required")
	}
	return s.accountRepo.GetAccountByID(accountID)
}

// GetStats retrieves account statistics.
func (s *Service) GetStats() (map[string]interface{}, error) {
	return s.accountRepo.GetStats()
}
