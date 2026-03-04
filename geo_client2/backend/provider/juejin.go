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

type JuejinProvider struct {
	*BaseProvider
	logger *logger.Logger
}

func NewJuejinProvider(headless bool, timeout int, accountID string) *JuejinProvider {
	return &JuejinProvider{
		BaseProvider: NewBaseProvider("juejin", headless, timeout, accountID),
		logger:       logger.GetLogger(),
	}
}

func (p *JuejinProvider) CheckLoginStatus() (bool, error) {
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
		p.logger.Warn("[CheckLoginStatus] juejin: WaitLoad failed: " + err.Error())
	}
	rod.Try(func() { _ = page.Timeout(5 * time.Second).WaitStable(1 * time.Second) })
	time.Sleep(2 * time.Second)

	info, _ := page.Info()
	if info != nil && strings.Contains(info.URL, "juejin.cn/login") {
		return false, nil
	}

	hasEditor, _, _ := page.Has("input.title-input")
	if hasEditor {
		return true, nil
	}

	return true, nil
}

func (p *JuejinProvider) Search(ctx context.Context, keyword, prompt string) (*SearchResult, error) {
	return nil, fmt.Errorf("search not supported for platform juejin")
}
