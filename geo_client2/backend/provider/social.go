package provider

import (
	"context"
	"fmt"
	"time"

	"geo_client2/backend/config"
	"geo_client2/backend/logger"
	"geo_client2/backend/scrape"

	"github.com/go-rod/rod"
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

	if err := page.WaitLoad(); err != nil {
		p.logger.Warn(fmt.Sprintf("[CheckLoginStatus] %s: WaitLoad failed: %s", p.platform, err.Error()))
	}
	rod.Try(func() { _ = page.Timeout(5 * time.Second).WaitStable(1 * time.Second) })
	time.Sleep(2 * time.Second)

	hasLogin, _, _ := page.HasR("a, button, div, span", "登录|注册|登录/注册")
	if hasLogin {
		p.logger.Debug(fmt.Sprintf("[CheckLoginStatus] %s: found login/register prompt", p.platform))
		return false, nil
	}

	return true, nil
}

func (p *SocialProvider) Search(ctx context.Context, keyword, prompt string) (*SearchResult, error) {
	if flow, version, err := scrape.LoadScrapeFlow(p.platform); err == nil && flow != nil {
		p.logger.InfoWithContext(ctx, "[SOCIAL-RPA] Loaded scrape flow", map[string]interface{}{"version": version, "platform": p.platform}, nil)
		browser, cleanup, err := p.LaunchBrowser(true)
		if err == nil {
			defer cleanup()
			page := browser.MustPage("")
			defer page.Close()
			runner := scrape.NewRunner(p.logger, p.platform)
			vars := map[string]string{"keyword": keyword, "prompt": prompt}
			if runErr := runner.Run(ctx, page, flow, vars); runErr == nil {
				res := runner.Result()
				return &SearchResult{Queries: res.Queries, Citations: convertCitations(res.Citations), FullText: res.FullText}, nil
			}
		}
	}
	return nil, fmt.Errorf("search not supported for platform %s", p.platform)
}
