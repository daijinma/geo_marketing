package provider

import (
	"context"
	"fmt"
)

// XiaohongshuProvider implements Xiaohongshu search.
type XiaohongshuProvider struct {
	*BaseProvider
}

// NewXiaohongshuProvider creates a new Xiaohongshu provider.
func NewXiaohongshuProvider(headless bool, timeout int, accountID string) *XiaohongshuProvider {
	return &XiaohongshuProvider{
		BaseProvider: NewBaseProvider("xiaohongshu", headless, timeout, accountID),
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

	hasLoginBtn, _, _ := page.HasR("button, div", "登录")
	if hasLoginBtn {
		return false, nil
	}

	hasRegisterBtn, _, _ := page.HasR("button, div", "注册")
	if hasRegisterBtn {
		return false, nil
	}

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
