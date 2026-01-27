package config

type PlatformConfig struct {
	Name     string
	LoginURL string
	HomeURL  string
	Category string
}

var Platforms = map[string]PlatformConfig{
	"doubao": {
		Name:     "Doubao",
		LoginURL: "https://www.doubao.com/",
		HomeURL:  "https://www.doubao.com/",
		Category: "ai_model",
	},
	"deepseek": {
		Name:     "DeepSeek",
		LoginURL: "https://chat.deepseek.com/sign_in",
		HomeURL:  "https://chat.deepseek.com/",
		Category: "ai_model",
	},
	"xiaohongshu": {
		Name:     "Xiaohongshu",
		LoginURL: "https://www.xiaohongshu.com/",
		HomeURL:  "https://www.xiaohongshu.com/",
		Category: "publishing",
	},
	"yiyan": {
		Name:     "Yiyan",
		LoginURL: "https://yiyan.baidu.com/",
		HomeURL:  "https://yiyan.baidu.com/",
		Category: "ai_model",
	},
	"yuanbao": {
		Name:     "Yuanbao",
		LoginURL: "https://yuanbao.tencent.com/login",
		HomeURL:  "https://yuanbao.tencent.com/",
		Category: "ai_model",
	},
}

func GetLoginURL(platform string) string {
	if config, ok := Platforms[platform]; ok {
		return config.LoginURL
	}
	return ""
}

func GetHomeURL(platform string) string {
	if config, ok := Platforms[platform]; ok {
		return config.HomeURL
	}
	return ""
}

func GetPlatformCategory(platform string) string {
	if config, ok := Platforms[platform]; ok {
		return config.Category
	}
	return "ai_model"
}

func IsAIModelPlatform(platform string) bool {
	return GetPlatformCategory(platform) == "ai_model"
}

func GetAIModelPlatforms() []string {
	platforms := []string{}
	for key, config := range Platforms {
		if config.Category == "ai_model" {
			platforms = append(platforms, key)
		}
	}
	return platforms
}

func GetPublishingPlatforms() []string {
	platforms := []string{}
	for key, config := range Platforms {
		if config.Category == "publishing" {
			platforms = append(platforms, key)
		}
	}
	return platforms
}
