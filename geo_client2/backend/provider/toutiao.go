package provider

import (
	"context"
	"fmt"
	"strings"
	"time"

	"geo_client2/backend/logger"

	"github.com/go-rod/rod"
)

// ToutiaoProvider handles login status check for 头条号 (Toutiao).
type ToutiaoProvider struct {
	*BaseProvider
	logger *logger.Logger
}

func NewToutiaoProvider(headless bool, timeout int, accountID string) *ToutiaoProvider {
	return &ToutiaoProvider{
		BaseProvider: NewBaseProvider("toutiao", headless, timeout, accountID),
		logger:       logger.GetLogger(),
	}
}

// CheckLoginStatus opens the 头条号 home page and checks whether the user is
// logged in. The page redirects to a login page when the session has expired.
func (p *ToutiaoProvider) CheckLoginStatus() (bool, error) {
	browser, cleanup, err := p.LaunchBrowser(true)
	if err != nil {
		return false, err
	}
	defer cleanup()
	defer p.Close()

	page := browser.MustPage("https://mp.toutiao.com/profile_v4/index")
	defer page.Close()

	if err := page.WaitLoad(); err != nil {
		p.logger.Warn("[CheckLoginStatus] toutiao: WaitLoad failed: " + err.Error())
	}
	rod.Try(func() { _ = page.Timeout(5 * time.Second).WaitStable(1 * time.Second) })
	time.Sleep(2 * time.Second)

	info := page.MustInfo()
	currentURL := ""
	if info != nil {
		currentURL = info.URL
	}

	// 未登录时会重定向到登录页面
	if strings.Contains(currentURL, "login") || strings.Contains(currentURL, "passport") {
		p.logger.Debug("[CheckLoginStatus] toutiao: redirected to login page, not logged in")
		return false, nil
	}

	// 登录后停留在 /profile_v4/
	if strings.Contains(currentURL, "profile_v4") {
		p.logger.Debug("[CheckLoginStatus] toutiao: on profile page, logged in")
		return true, nil
	}

	// 备用：检查发布入口元素
	hasPublishEntry, _, _ := page.HasR("div, a, button", "发文|发布文章|写文章|创作")
	if hasPublishEntry {
		p.logger.Debug("[CheckLoginStatus] toutiao: found publish entry, logged in")
		return true, nil
	}

	hasLoginBtn, _, _ := page.HasR("button, a", "登录|注册|手机号")
	if hasLoginBtn {
		p.logger.Debug("[CheckLoginStatus] toutiao: found login button, not logged in")
		return false, nil
	}

	p.logger.Debug("[CheckLoginStatus] toutiao: no clear indicators, assuming logged in")
	return true, nil
}

func (p *ToutiaoProvider) Search(ctx context.Context, keyword, prompt string) (*SearchResult, error) {
	return nil, fmt.Errorf("search not supported for platform toutiao")
}
