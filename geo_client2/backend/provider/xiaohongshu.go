package provider

import (
	"context"
	"fmt"

	"github.com/go-rod/rod"
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
	browser, _, err := p.LaunchBrowser(true)
	if err != nil {
		return false, err
	}

	page := browser.MustPage("https://www.xiaohongshu.com/")
	defer page.Close()

	hasProfile := false
	err = rod.Try(func() {
		page.MustElement(".user-name")
		hasProfile = true
	})

	return hasProfile, nil
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
