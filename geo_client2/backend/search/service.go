package search

import (
	"fmt"
	"geo_client2/backend/database/repositories"
	"geo_client2/backend/logger"
	"geo_client2/backend/provider"
	"geo_client2/backend/task"
)

// Service handles search operations.
type Service struct {
	taskManager  *task.Manager
	providerFact *provider.Factory
	loginRepo    *repositories.LoginStatusRepository
	accountRepo  *repositories.AccountRepository
	settingsRepo *repositories.SettingsRepository
}

// NewService creates a new search service.
func NewService(taskManager *task.Manager, providerFact *provider.Factory, loginRepo *repositories.LoginStatusRepository, accountRepo *repositories.AccountRepository, settingsRepo *repositories.SettingsRepository) *Service {
	return &Service{
		taskManager:  taskManager,
		providerFact: providerFact,
		loginRepo:    loginRepo,
		accountRepo:  accountRepo,
		settingsRepo: settingsRepo,
	}
}

func (s *Service) CreateTask(keywords, platforms []string, queryCount int) (int64, error) {
	for _, p := range platforms {
		activeAccount, err := s.accountRepo.GetActiveAccount(p)
		if err != nil {
			return 0, fmt.Errorf("failed to get active account for %s: %w", p, err)
		}
		if activeAccount == nil {
			return 0, fmt.Errorf("no active account found for platform %s", p)
		}
	}

	headlessStr, _ := s.settingsRepo.Get("browser_headless")
	headless := headlessStr != "false"

	taskSettings := map[string]interface{}{
		"headless": headless,
	}

	return s.taskManager.CreateLocalSearchTask(keywords, platforms, queryCount, "local_search", "local", nil, taskSettings)
}

// CheckLoginStatus checks login status for a platform using active account.
func (s *Service) CheckLoginStatus(platform string) (bool, error) {
	// Get active account for platform
	activeAccount, err := s.accountRepo.GetActiveAccount(platform)
	if err != nil {
		return false, fmt.Errorf("failed to get active account: %w", err)
	}
	if activeAccount == nil {
		return false, nil // No account means not logged in
	}

	prov, err := s.providerFact.GetProvider(platform, true, 60000, activeAccount.AccountID)
	if err != nil {
		return false, err
	}
	loggedIn, err := prov.CheckLoginStatus()
	if err != nil {
		return false, err
	}

	// Update login status in DB
	err = s.loginRepo.UpdateLoginStatus(platform, activeAccount.AccountID, loggedIn)
	if err != nil {
		logger.GetLogger().Error("Failed to update login status", err)
	}

	return loggedIn, nil
}
