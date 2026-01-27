package auth

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"geo_client2/backend/database/repositories"
)

// Service handles authentication.
type Service struct {
	authRepo *repositories.AuthRepository
}

// NewService creates a new auth service.
func NewService(authRepo *repositories.AuthRepository) *Service {
	return &Service{authRepo: authRepo}
}

// LoginResponse represents the login API response.
type LoginResponse struct {
	Success   bool   `json:"success"`
	Token     string `json:"token"`
	ExpiresAt string `json:"expires_at"`
}

// Login performs login and saves the token.
func (s *Service) Login(username, password, apiBaseURL string) (*LoginResponse, error) {
	loginURL := fmt.Sprintf("%s/client/auth/login", apiBaseURL)

	reqBody, _ := json.Marshal(map[string]string{
		"username": username,
		"password": password,
	})

	resp, err := http.Post(loginURL, "application/json", bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, fmt.Errorf("login request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		return nil, fmt.Errorf("登录失败：用户名或密码错误")
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("登录失败: HTTP %d", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	var loginResp LoginResponse
	if err := json.Unmarshal(body, &loginResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if !loginResp.Success {
		return nil, fmt.Errorf("登录失败：用户名或密码错误")
	}

	// Save token
	if err := s.authRepo.SaveToken(loginResp.Token, loginResp.ExpiresAt); err != nil {
		return nil, fmt.Errorf("failed to save token: %w", err)
	}

	return &loginResp, nil
}

// GetToken retrieves the stored token.
func (s *Service) GetToken() (token, expiresAt string, isValid bool, err error) {
	token, expiresAt, err = s.authRepo.GetToken()
	if err != nil || token == "" {
		return "", "", false, err
	}
	expired, err := s.authRepo.IsTokenExpired()
	if err != nil {
		return token, expiresAt, false, err
	}
	return token, expiresAt, !expired, nil
}

// Logout deletes the stored token.
func (s *Service) Logout() error {
	return s.authRepo.DeleteToken()
}

// CheckTokenValid checks if the token is valid.
func (s *Service) CheckTokenValid() (bool, error) {
	expired, err := s.authRepo.IsTokenExpired()
	return !expired, err
}
