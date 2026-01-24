package search

import (
	"fmt"
	"geo_client2/backend/database/repositories"
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

// CreateTask creates a search task.
func (s *Service) CreateTask(keywords, platforms []string, queryCount int) (int64, error) {
	// Check login status for all platforms using active accounts
	for _, p := range platforms {
		// Get active account for platform
		activeAccount, err := s.accountRepo.GetActiveAccount(p)
		if err != nil {
			return 0, fmt.Errorf("failed to get active account for %s: %w", p, err)
		}
		if activeAccount == nil {
			return 0, fmt.Errorf("no active account found for platform %s", p)
		}

		prov, err := s.providerFact.GetProvider(p, true, 60000, activeAccount.AccountID)
		if err != nil {
			return 0, err
		}
		loggedIn, err := prov.CheckLoginStatus()
		if err != nil || !loggedIn {
			return 0, fmt.Errorf("platform %s not logged in", p)
		}
	}

	// Get headless setting
	headlessStr, _ := s.settingsRepo.Get("browser_headless")
	headless := headlessStr != "false"

	// Create task with runtime settings
	// Note: We're passing settings to the task manager, which should ideally pass them to the executor.
	// However, the executor is initialized with a provider factory that has a fixed headless setting.
	// We might need to update the executor/provider logic to accept overrides or refresh settings.
	// For now, let's assume the provider factory's default is used, BUT the App startup logic now reads the setting.
	// To make it dynamic without restarting, we need to pass the setting here.

	// Actually, the provider factory is a singleton-ish in App.
	// If we want per-task dynamic headless, we should update the task creation to include this preference,
	// or have the executor read the setting at runtime.

	// Let's pass the headless setting as part of the task metadata or handle it in the executor.
	// Current executor implementation likely uses the provider factory it was given.
	// To support dynamic switching, we'll pass the setting value in the 'settings' map of the task.

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
		fmt.Printf("Failed to update login status: %v\n", err)
	}

	return loggedIn, nil
}
