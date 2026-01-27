package provider

import (
	"context"
	"fmt"
	"strings"
	"time"

	"geo_client2/backend/logger"
)

// XiaohongshuProvider implements Xiaohongshu search.
type XiaohongshuProvider struct {
	*BaseProvider
	logger *logger.Logger
}

// NewXiaohongshuProvider creates a new Xiaohongshu provider.
func NewXiaohongshuProvider(headless bool, timeout int, accountID string) *XiaohongshuProvider {
	return &XiaohongshuProvider{
		BaseProvider: NewBaseProvider("xiaohongshu", headless, timeout, accountID),
		logger:       logger.GetLogger(),
	}
}

// GetLoginUrl returns Xiaohongshu login URL.
func (p *XiaohongshuProvider) GetLoginUrl() string {
	return p.loginURL
}

func (p *XiaohongshuProvider) CheckLoginStatus() (bool, error) {
	browser, cleanup, err := p.LaunchBrowser(true)
	if err != nil {
		return false, err
	}
	defer cleanup()
	defer p.Close()

	page := browser.MustPage("https://www.xiaohongshu.com/")
	defer page.Close()
	page.MustWaitLoad()
	time.Sleep(3 * time.Second)

	// Strategy 1: Check Cookies
	cookies, err := page.Cookies([]string{"https://www.xiaohongshu.com"})
	if err == nil && len(cookies) > 0 {
		hasAuthCookie := false
		for _, cookie := range cookies {
			cookieName := strings.ToLower(cookie.Name)
			if strings.Contains(cookieName, "session") ||
				strings.Contains(cookieName, "token") ||
				strings.Contains(cookieName, "auth") ||
				strings.Contains(cookieName, "access_token") ||
				strings.Contains(cookieName, "user_id") ||
				cookieName == "jsessionid" ||
				cookieName == "sid" ||
				strings.Contains(cookieName, "web_session") {
				hasAuthCookie = true
				p.logger.Debug(fmt.Sprintf("[CheckLoginStatus] Xiaohongshu: Found auth cookie: %s", cookie.Name))
				break
			}
		}
		if hasAuthCookie {
			p.logger.Debug("[CheckLoginStatus] Xiaohongshu: Valid auth cookie found, likely logged in")
		} else {
			p.logger.Debug("[CheckLoginStatus] Xiaohongshu: No auth cookies found, likely not logged in")
		}
	}

	// Strategy 2: Negative Element Detection (login/register button)
	hasLoginBtn, _, _ := page.HasR("button, div", "登录")
	if hasLoginBtn {
		p.logger.Debug("[CheckLoginStatus] Xiaohongshu: Found login button, not logged in")
		return false, nil
	}

	hasRegisterBtn, _, _ := page.HasR("button, div", "注册")
	if hasRegisterBtn {
		p.logger.Debug("[CheckLoginStatus] Xiaohongshu: Found register button, not logged in")
		return false, nil
	}

	// Strategy 3: HTTP Response Check
	bodyText, err := page.Element("body")
	if err == nil {
		text, _ := bodyText.Text()
		textLower := strings.ToLower(text)

		unauthorizedKeywords := []string{
			"unauthorized", "unauthenticated", "login required", "sign in required",
			"access denied", "invalid session", "token expired",
			"未登录", "请登录", "需要登录", "登录已过期", "未授权", "会话已过期", "令牌无效",
		}

		for _, keyword := range unauthorizedKeywords {
			if strings.Contains(textLower, keyword) {
				p.logger.Debug(fmt.Sprintf("[CheckLoginStatus] Xiaohongshu: Found unauthorized keyword '%s', not logged in", keyword))
				return false, nil
			}
		}
	}

	// If we reach here with auth cookies and no negative indicators, assume logged in
	if err == nil && len(cookies) > 0 {
		for _, cookie := range cookies {
			cookieName := strings.ToLower(cookie.Name)
			if strings.Contains(cookieName, "session") ||
				strings.Contains(cookieName, "token") ||
				strings.Contains(cookieName, "auth") ||
				strings.Contains(cookieName, "web_session") {
				p.logger.Debug("[CheckLoginStatus] Xiaohongshu: Auth cookie present and no negative indicators found, assuming logged in")
				return true, nil
			}
		}
	}

	p.logger.Debug("[CheckLoginStatus] Xiaohongshu: No negative indicators found, likely logged in")
	return true, nil
}

func (p *XiaohongshuProvider) Search(ctx context.Context, keyword, prompt string) (*SearchResult, error) {
	browser, _, err := p.LaunchBrowser(false)
	if err != nil {
		return nil, err
	}

	page := browser.MustPage("https://www.xiaohongshu.com/")
	page.MustWaitLoad()

	return &SearchResult{
		Queries:   []string{keyword},
		Citations: []Citation{},
		FullText:  fmt.Sprintf("Xiaohongshu mock search result for: %s. Browser opened successfully.", keyword),
	}, nil
}
