package settings

import "geo_client2/backend/database/repositories"

// Service handles settings.
type Service struct {
	settingsRepo *repositories.SettingsRepository
}

// NewService creates a new settings service.
func NewService(settingsRepo *repositories.SettingsRepository) *Service {
	return &Service{settingsRepo: settingsRepo}
}

// Get retrieves a setting value.
func (s *Service) Get(key string) (string, error) {
	return s.settingsRepo.Get(key)
}

// Set saves a setting value.
func (s *Service) Set(key, value string) error {
	return s.settingsRepo.Save(key, value)
}
