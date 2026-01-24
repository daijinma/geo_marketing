package config

// PlatformConfig holds configuration for a search platform.
type PlatformConfig struct {
	Name     string
	LoginURL string
	HomeURL  string
}

// Platforms holds the configuration for all supported platforms.
var Platforms = map[string]PlatformConfig{
	"doubao": {
		Name:     "Doubao",
		LoginURL: "https://www.doubao.com/",
		HomeURL:  "https://www.doubao.com/",
	},
	"deepseek": {
		Name:     "DeepSeek",
		LoginURL: "https://chat.deepseek.com/sign_in",
		HomeURL:  "https://chat.deepseek.com/",
	},
	"xiaohongshu": {
		Name:     "Xiaohongshu",
		LoginURL: "https://www.xiaohongshu.com/",
		HomeURL:  "https://www.xiaohongshu.com/",
	},
}

// GetLoginURL returns the login URL for a platform.
func GetLoginURL(platform string) string {
	if config, ok := Platforms[platform]; ok {
		return config.LoginURL
	}
	return ""
}

// GetHomeURL returns the home URL for a platform.
func GetHomeURL(platform string) string {
	if config, ok := Platforms[platform]; ok {
		return config.HomeURL
	}
	return ""
}
