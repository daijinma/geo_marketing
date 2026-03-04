package provider

import (
	"context"
	"fmt"
	"strings"
	"time"

	"geo_client2/backend/config"
	"geo_client2/backend/logger"

	"github.com/go-rod/rod"
)

type CnblogsProvider struct {
	*BaseProvider
	logger *logger.Logger
}

func NewCnblogsProvider(headless bool, timeout int, accountID string) *CnblogsProvider {
	return &CnblogsProvider{
		BaseProvider: NewBaseProvider("cnblogs", headless, timeout, accountID),
		logger:       logger.GetLogger(),
	}
}

func (p *CnblogsProvider) CheckLoginStatus() (bool, error) {
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
		p.logger.Warn("[CheckLoginStatus] cnblogs: WaitLoad failed: " + err.Error())
	}
	rod.Try(func() { _ = page.Timeout(5 * time.Second).WaitStable(1 * time.Second) })
	time.Sleep(2 * time.Second)

	info, _ := page.Info()
	if info != nil && (strings.Contains(info.URL, "account.cnblogs.com/signin") || strings.Contains(info.URL, "login")) {
		return false, nil
	}

	hasEditor, _, _ := page.Has("#post-title")
	if hasEditor {
		return true, nil
	}

	return true, nil
}

func (p *CnblogsProvider) Search(ctx context.Context, keyword, prompt string) (*SearchResult, error) {
	return nil, fmt.Errorf("search not supported for platform cnblogs")
}
