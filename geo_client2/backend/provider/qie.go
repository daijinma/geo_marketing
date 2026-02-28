package provider

import (
	"context"
	"fmt"
	"time"

	"geo_client2/backend/config"
	"geo_client2/backend/logger"

	"github.com/go-rod/rod"
)

type QieProvider struct {
	*BaseProvider
	logger *logger.Logger
}

func NewQieProvider(headless bool, timeout int, accountID string) *QieProvider {
	return &QieProvider{
		BaseProvider: NewBaseProvider("qie", headless, timeout, accountID),
		logger:       logger.GetLogger(),
	}
}

func (p *QieProvider) CheckLoginStatus() (bool, error) {
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
		p.logger.Warn("[CheckLoginStatus] qie: WaitLoad failed: " + err.Error())
	}
	rod.Try(func() { _ = page.Timeout(5 * time.Second).WaitStable(1 * time.Second) })
	time.Sleep(2 * time.Second)

	hasPublishBtn, _, _ := page.HasR("button, a", "发布|发表|创作")
	if hasPublishBtn {
		p.logger.Debug("[CheckLoginStatus] qie: found publish button, user is logged in")
		return true, nil
	}

	hasAvatar, _, _ := page.Has(".avatar, [class*='avatar'], [class*='user-info']")
	if hasAvatar {
		p.logger.Debug("[CheckLoginStatus] qie: found avatar, user is logged in")
		return true, nil
	}

	hasLoginBtn, _, _ := page.HasR("button, a, div", "登录|注册|QQ登录|微信登录")
	if hasLoginBtn {
		p.logger.Debug("[CheckLoginStatus] qie: found login button, not logged in")
		return false, nil
	}

	p.logger.Debug("[CheckLoginStatus] qie: no clear indicators, assuming logged in")
	return true, nil
}

func (p *QieProvider) Search(ctx context.Context, keyword, prompt string) (*SearchResult, error) {
	return nil, fmt.Errorf("search not supported for platform qie")
}
