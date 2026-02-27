package provider

import (
	"context"
	"fmt"
	"time"

	"geo_client2/backend/config"
	"geo_client2/backend/logger"
)

// SocialProvider provides basic login support for social media platforms.
type SocialProvider struct {
	*BaseProvider
	logger *logger.Logger
}

// NewSocialProvider creates a new social media provider.
func NewSocialProvider(platform string, headless bool, timeout int, accountID string) *SocialProvider {
	return &SocialProvider{
		BaseProvider: NewBaseProvider(platform, headless, timeout, accountID),
		logger:       logger.GetLogger(),
	}
}

func (p *SocialProvider) CheckLoginStatus() (bool, error) {
	browser, cleanup, err := p.LaunchBrowser(true)
	if err != nil {
		return false, err
	}
	defer cleanup()
	defer p.Close()

	homeURL := config.GetHomeURL(p.platform)
	if homeURL == "" {
		homeURL = p.loginURL
	}

	page := browser.MustPage(homeURL)
	defer page.Close()
	page.MustWaitLoad()
	time.Sleep(2 * time.Second)

	hasLogin, _, _ := page.HasR("a, button, div, span", "登录|注册|登录/注册")
	if hasLogin {
		p.logger.Debug(fmt.Sprintf("[CheckLoginStatus] %s: found login/register prompt", p.platform))
		return false, nil
	}

	return true, nil
}

func (p *SocialProvider) Search(ctx context.Context, keyword, prompt string) (*SearchResult, error) {
	return nil, fmt.Errorf("search not supported for platform %s", p.platform)
}
