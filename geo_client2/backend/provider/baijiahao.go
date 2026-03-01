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

type BaijiahaoProvider struct {
	*BaseProvider
	logger *logger.Logger
}

func NewBaijiahaoProvider(headless bool, timeout int, accountID string) *BaijiahaoProvider {
	return &BaijiahaoProvider{
		BaseProvider: NewBaseProvider("baijiahao", headless, timeout, accountID),
		logger:       logger.GetLogger(),
	}
}

func (p *BaijiahaoProvider) CheckLoginStatus() (bool, error) {
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

	// Use error-handling versions to avoid panic on context destruction
	if err := page.WaitLoad(); err != nil {
		p.logger.Warn("[CheckLoginStatus] baijiahao: WaitLoad failed: " + err.Error())
		// Continue anyway - page might still be usable
	}
	rod.Try(func() { _ = page.Timeout(5 * time.Second).WaitStable(1 * time.Second) })
	time.Sleep(2 * time.Second)

	hasPublishBtn, _, _ := page.HasR("button, a, div", "发布|写文章|创作")
	if hasPublishBtn {
		p.logger.Debug("[CheckLoginStatus] baijiahao: found publish button, user is logged in")
		return true, nil
	}

	hasAvatar, _, _ := page.Has(".avatar, [class*='avatar'], [class*='user']")
	if hasAvatar {
		p.logger.Debug("[CheckLoginStatus] baijiahao: found avatar, user is logged in")
		return true, nil
	}

	hasLoginBtn, _, _ := page.HasR("button, a, div", "登录|注册|百度账号登录")
	if hasLoginBtn {
		p.logger.Debug("[CheckLoginStatus] baijiahao: found login button, not logged in")
		return false, nil
	}

	p.logger.Debug("[CheckLoginStatus] baijiahao: no clear indicators, assuming logged in")
	return true, nil
}

func (p *BaijiahaoProvider) Search(ctx context.Context, keyword, prompt string) (*SearchResult, error) {
	if flow, version, err := scrape.LoadScrapeFlow("baijiahao"); err == nil && flow != nil {
		p.logger.InfoWithContext(ctx, "[BAIJIAHAO-RPA] Loaded scrape flow", map[string]interface{}{"version": version}, nil)
		browser, cleanup, err := p.LaunchBrowser(true)
		if err == nil {
			defer cleanup()
			page := browser.MustPage("")
			defer page.Close()
			runner := scrape.NewRunner(p.logger, "baijiahao")
			vars := map[string]string{"keyword": keyword, "prompt": prompt}
			if runErr := runner.Run(ctx, page, flow, vars); runErr == nil {
				res := runner.Result()
				return &SearchResult{Queries: res.Queries, Citations: convertCitations(res.Citations), FullText: res.FullText}, nil
			}
		}
	}
	return nil, fmt.Errorf("search not supported for platform baijiahao")
}
