package provider

import (
	"context"
	"fmt"
	"time"

	"geo_client2/backend/config"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/input"
)

// DoubaoProvider implements Doubao search.
type DoubaoProvider struct {
	*BaseProvider
}

// NewDoubaoProvider creates a new Doubao provider.
func NewDoubaoProvider(headless bool, timeout int, accountID string) *DoubaoProvider {
	return &DoubaoProvider{
		BaseProvider: NewBaseProvider("doubao", headless, timeout, accountID),
	}
}

// GetLoginUrl returns Doubao login URL.
func (d *DoubaoProvider) GetLoginUrl() string {
	return d.loginURL
}

// CheckLoginStatus checks if logged in.
func (d *DoubaoProvider) CheckLoginStatus() (bool, error) {
	browser, cleanup, err := d.LaunchBrowser(true) // Headless
	if err != nil {
		return false, err
	}
	defer cleanup()

	page := browser.MustPage(d.GetLoginUrl())
	page.MustWaitLoad()

	// Wait a bit for dynamic content
	time.Sleep(2 * time.Second)

	// Check for "登录" (Login) button. If it exists, we are NOT logged in.
	// Using Has method with a text selector
	hasLoginBtn, _, _ := page.HasR("button", "登录")
	if hasLoginBtn {
		return false, nil
	}

	// Also check for "注册" (Register) just in case
	hasRegisterBtn, _, _ := page.HasR("button", "注册")
	if hasRegisterBtn {
		return false, nil
	}

	// If no login/register buttons found, assume logged in
	return true, nil
}

// Search performs a search.
func (d *DoubaoProvider) Search(ctx context.Context, keyword, prompt string) (*SearchResult, error) {
	browser, cleanup, err := d.LaunchBrowser(d.headless)
	if err != nil {
		return nil, err
	}
	defer cleanup()

	// 1. Go to home page
	homeURL := config.GetHomeURL("doubao")
	page := browser.Context(ctx).MustPage(homeURL)
	defer page.Close()

	// Wait for stability
	page.MustWaitStable()

	// 2. Find input box and type
	// Doubao usually has a textarea
	textarea, err := page.Element("textarea")
	if err != nil {
		return nil, fmt.Errorf("failed to find textarea: %w", err)
	}

	// Click and input
	textarea.MustClick().MustInput(keyword)

	// 3. Submit
	// Doubao usually uses Enter or a send button
	page.KeyActions().Press(input.Enter).Release(input.Enter).MustDo()

	// 4. Wait for answer
	// Doubao's streaming response: wait for markdown to appear and then wait for idle
	err = rod.Try(func() {
		// Wait for any message content
		page.MustWaitElementsMoreThan("div[class*=\"message-content\"]", 0)

		page.MustWaitIdle()
		time.Sleep(3 * time.Second)
	})
	if err != nil {
		time.Sleep(10 * time.Second)
	}

	// 5. Extract text
	var fullText string
	messages := page.MustElements("div[class*=\"message-content\"]")
	if len(messages) > 0 {
		// Get last assistant message
		fullText = messages[len(messages)-1].MustText()
	} else {
		// Fallback
		fullText = page.MustElement("body").MustText()
	}

	return &SearchResult{
		Queries:   []string{keyword},
		Citations: []Citation{},
		FullText:  fullText,
	}, nil
}
