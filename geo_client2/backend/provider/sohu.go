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

type SohuProvider struct {
	*BaseProvider
	logger *logger.Logger
}

func NewSohuProvider(headless bool, timeout int, accountID string) *SohuProvider {
	return &SohuProvider{
		BaseProvider: NewBaseProvider("sohu", headless, timeout, accountID),
		logger:       logger.GetLogger(),
	}
}

func (p *SohuProvider) CheckLoginStatus() (bool, error) {
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
		p.logger.Warn("[CheckLoginStatus] sohu: WaitLoad failed: " + err.Error())
	}
	rod.Try(func() { _ = page.Timeout(5 * time.Second).WaitStable(1 * time.Second) })
	time.Sleep(2 * time.Second)

	hasPublishBtn, _, _ := page.HasR("button", "发布内容")
	if hasPublishBtn {
		p.logger.Debug("[CheckLoginStatus] sohu: found publish button, user is logged in")
		return true, nil
	}

	hasAvatar, _, _ := page.Has(".user-avatar, .avatar, [class*='avatar']")
	if hasAvatar {
		p.logger.Debug("[CheckLoginStatus] sohu: found avatar, user is logged in")
		return true, nil
	}

	hasLoginBtn, _, _ := page.HasR("button, a", "登录|注册")
	if hasLoginBtn {
		p.logger.Debug("[CheckLoginStatus] sohu: found login/register button, not logged in")
		return false, nil
	}

	p.logger.Debug("[CheckLoginStatus] sohu: no clear indicators, assuming not logged in")
	return false, nil
}

func (p *SohuProvider) Search(ctx context.Context, keyword, prompt string) (*SearchResult, error) {
	if flow, version, err := scrape.LoadScrapeFlow("sohu"); err == nil && flow != nil {
		p.logger.InfoWithContext(ctx, "[SOHU-RPA] Loaded scrape flow", map[string]interface{}{"version": version}, nil)
		browser, cleanup, err := p.LaunchBrowser(true)
		if err == nil {
			defer cleanup()
			page := browser.MustPage("")
			defer page.Close()
			runner := scrape.NewRunner(p.logger, "sohu")
			vars := map[string]string{"keyword": keyword, "prompt": prompt}
			if runErr := runner.Run(ctx, page, flow, vars); runErr == nil {
				res := runner.Result()
				return &SearchResult{Queries: res.Queries, Citations: convertCitations(res.Citations), FullText: res.FullText}, nil
			}
		}
	}
	return nil, fmt.Errorf("search not supported for platform sohu")
}
