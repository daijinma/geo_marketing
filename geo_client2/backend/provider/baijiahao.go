package provider

import (
	"context"
	"fmt"
	"time"

	"geo_client2/backend/config"
	"geo_client2/backend/logger"
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
	page.MustWaitLoad()
	page.MustWaitIdle()
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
	return nil, fmt.Errorf("search not supported for platform baijiahao")
}
