package provider

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"geo_client2/backend/config"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
)

// SearchResult represents a search result.
type SearchResult struct {
	Queries   []string   `json:"queries"`
	Citations []Citation `json:"citations"`
	FullText  string     `json:"full_text"`
}

// Citation represents a citation.
type Citation struct {
	URL          string `json:"url"`
	Domain       string `json:"domain"`
	Title        string `json:"title"`
	Snippet      string `json:"snippet"`
	QueryIndexes []int  `json:"query_indexes"`
	Query        string `json:"query"`      // 根据 QueryIndexes[0] 从 Queries 数组中获取的查询词
	CiteIndex    int    `json:"cite_index"` // 引用序号，用于排序
}

// Provider interface for search providers.
type Provider interface {
	Search(ctx context.Context, keyword, prompt string) (*SearchResult, error)
	CheckLoginStatus() (bool, error)
	GetLoginUrl() string
	StartLogin() (func(), error)
	Close() error
}

// BaseProvider provides common functionality.
type BaseProvider struct {
	userDataDir string
	headless    bool
	timeout     int
	accountID   string // Account ID for multi-account support
	platform    string // Platform name
	loginURL    string
	browser     *rod.Browser
}

// NewBaseProvider creates a new base provider.
// If accountID is empty, uses the old structure (for backward compatibility during migration).
func NewBaseProvider(platform string, headless bool, timeout int, accountID string) *BaseProvider {
	homeDir, _ := os.UserHomeDir()
	var userDataDir string
	if accountID != "" {
		// New structure: ~/.geo_client2/browser_data/{platform}/{account_id}/
		userDataDir = filepath.Join(homeDir, ".geo_client2", "browser_data", platform, accountID)
	} else {
		// Old structure (for backward compatibility): ~/.geo_client2/browser_data/{platform}/
		userDataDir = filepath.Join(homeDir, ".geo_client2", "browser_data", platform)
	}

	loginURL := config.GetLoginURL(platform)

	return &BaseProvider{
		userDataDir: userDataDir,
		headless:    headless,
		timeout:     timeout,
		accountID:   accountID,
		platform:    platform,
		loginURL:    loginURL,
	}
}

// clearChromeLockFiles 删除 Chrome/Chromium 在 UserDataDir 下的单实例锁文件，避免
// "Failed to get the debug url: 正在现有的浏览器会话中打开"：当存在残留锁或已有实例
// 占用同一 profile 时，新进程会尝试在现有会话中打开并退出，不输出 ws 调试地址。
func clearChromeLockFiles(userDataDir string) {
	for _, name := range []string{"SingletonLock", "SingletonSocket", "SingletonCookie"} {
		p := filepath.Join(userDataDir, name)
		_ = os.Remove(p)
	}
}

func (b *BaseProvider) LaunchBrowser(headless bool) (*rod.Browser, func(), error) {
	if b.browser != nil {
		return b.browser, func() {}, nil
	}

	if b.userDataDir != "" {
		if err := os.MkdirAll(b.userDataDir, 0755); err != nil {
			return nil, nil, fmt.Errorf("creating user data dir: %w", err)
		}
		clearChromeLockFiles(b.userDataDir)
	}

	l := launcher.New().
		UserDataDir(b.userDataDir).
		Headless(headless).
		Set("lang", "zh-CN") // Set language to Chinese

	l.Set("disable-blink-features", "AutomationControlled")
	l.NoSandbox(true)
	l.Set("disable-dev-shm-usage")
	l.Set("disable-gpu")

	u, err := l.Launch()
	if err != nil {
		return nil, nil, fmt.Errorf("launching browser: %w", err)
	}

	browser := rod.New().ControlURL(u)
	err = browser.Connect()
	if err != nil {
		return nil, nil, fmt.Errorf("connecting to browser: %w", err)
	}
	// 禁用默认设备模拟（修复窗口调整大小时内容不随之调整的问题）
	browser = browser.NoDefaultDevice()

	b.browser = browser

	cleanup := func() {}

	return browser, cleanup, nil
}

func (b *BaseProvider) Close() error {
	if b.browser != nil {
		err := b.browser.Close()
		b.browser = nil
		return err
	}
	return nil
}

// StartLogin opens a non-headless browser for the user to login.
// Returns a cleanup function that closes the browser.
func (b *BaseProvider) StartLogin() (func(), error) {
	if b.loginURL == "" {
		return nil, fmt.Errorf("login URL not defined for platform %s", b.platform)
	}

	// Always non-headless for login
	browser, cleanup, err := b.LaunchBrowser(false)
	if err != nil {
		return nil, err
	}

	// Create a new page and navigate to login URL
	page := browser.MustPage(b.loginURL)
	page.MustWaitLoad()

	// Return the cleanup function
	return cleanup, nil
}

// GetLoginUrl returns the login URL.
func (b *BaseProvider) GetLoginUrl() string {
	return b.loginURL
}

// extractDomain extracts domain from URL.
func (b *BaseProvider) extractDomain(url string) string {
	// Simplified - use a proper URL parser in production
	return url
}

// Factory creates providers.
type Factory struct {
	headless bool
	timeout  int
}

// NewFactory creates a new provider factory.
func NewFactory(headless bool, timeout int) *Factory {
	return &Factory{headless: headless, timeout: timeout}
}

func (f *Factory) SetHeadless(headless bool) {
	f.headless = headless
}

func (f *Factory) IsHeadless() bool {
	return f.headless
}

// GetProvider returns a provider for the given platform and account.
// If accountID is empty, it will use the active account for the platform.
func (f *Factory) GetProvider(platform string, headless bool, timeout int, accountID string) (Provider, error) {
	switch platform {
	case "doubao":
		return NewDoubaoProvider(headless, timeout, accountID), nil
	case "deepseek":
		return NewDeepSeekProvider(headless, timeout, accountID), nil
	case "xiaohongshu":
		return NewXiaohongshuProvider(headless, timeout, accountID), nil
	case "yiyan":
		return NewYiyanProvider(headless, timeout, accountID), nil
	case "yuanbao":
		return NewYuanbaoProvider(headless, timeout, accountID), nil
	default:
		return nil, fmt.Errorf("unsupported platform: %s", platform)
	}
}
