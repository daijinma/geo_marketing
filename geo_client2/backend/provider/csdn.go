package provider

import (
	"context"
	"fmt"
	"time"

	"geo_client2/backend/config"
	"geo_client2/backend/logger"

	"github.com/go-rod/rod"
)

type CsdnProvider struct {
	*BaseProvider
	logger *logger.Logger
}

func NewCsdnProvider(headless bool, timeout int, accountID string) *CsdnProvider {
	return &CsdnProvider{
		BaseProvider: NewBaseProvider("csdn", headless, timeout, accountID),
		logger:       logger.GetLogger(),
	}
}

func (p *CsdnProvider) CheckLoginStatus() (bool, error) {
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
		p.logger.Warn("[CheckLoginStatus] csdn: WaitLoad failed: " + err.Error())
	}
	rod.Try(func() { _ = page.Timeout(5 * time.Second).WaitStable(1 * time.Second) })
	time.Sleep(2 * time.Second)

	hasCreateLink, createEl, _ := page.HasR("a", "创作")
	if hasCreateLink && createEl != nil {
		href, err := createEl.Attribute("href")
		if err == nil && href != nil && *href == "https://mp.csdn.net/edit" {
			p.logger.Debug("[CheckLoginStatus] csdn: found '创作' link with correct href, user is logged in")
			return true, nil
		}
	}

	hasLoginBtn, _, _ := page.HasR("a, button, div, span", "登录|注册|登录/注册")
	if hasLoginBtn {
		p.logger.Debug("[CheckLoginStatus] csdn: found login/register prompt, not logged in")
		return false, nil
	}

	hasAvatar, _, _ := page.Has("img.avatar, .user-avatar, .toolbar-avatar, [class*='avatar']")
	if hasAvatar {
		p.logger.Debug("[CheckLoginStatus] csdn: found avatar element, user is logged in")
		return true, nil
	}

	hasUserDropdown, _, _ := page.Has(".toolbar-user, .user-dropdown, [class*='user-info']")
	if hasUserDropdown {
		p.logger.Debug("[CheckLoginStatus] csdn: found user dropdown, user is logged in")
		return true, nil
	}

	p.logger.Debug("[CheckLoginStatus] csdn: no clear login indicators found, assuming logged in")
	return true, nil
}

func (p *CsdnProvider) Search(ctx context.Context, keyword, prompt string) (*SearchResult, error) {
	return nil, fmt.Errorf("search not supported for platform csdn")
}
