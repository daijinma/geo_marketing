package config

type PlatformConfig struct {
	Name     string
	LoginURL string
	HomeURL  string
	Category string
}

var Platforms = map[string]PlatformConfig{
	// AI 大模型
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
	// 社交媒体发布平台
	"xiaohongshu": {
		Name:     "小红书",
		LoginURL: "https://www.xiaohongshu.com/",
		HomeURL:  "https://www.xiaohongshu.com/",
		Category: "social_media",
	},
	"zhihu": {
		Name:     "知乎",
		LoginURL: "https://www.zhihu.com/signin",
		HomeURL:  "https://www.zhihu.com",
		Category: "social_media",
	},
	"sohu": {
		Name:     "搜狐号",
		LoginURL: "https://mp.sohu.com/mpfe/v4/login",
		HomeURL:  "https://mp.sohu.com",
		Category: "social_media",
	},
	"csdn": {
		Name:     "CSDN",
		LoginURL: "https://passport.csdn.net/login",
		HomeURL:  "https://www.csdn.net",
		Category: "social_media",
	},
	"qie": {
		Name:     "企鹅号",
		LoginURL: "https://om.qq.com/userAuth/index",
		HomeURL:  "https://om.qq.com",
		Category: "social_media",
	},
	"baijiahao": {
		Name:     "百家号",
		LoginURL: "https://baijiahao.baidu.com/builder/theme/bjh/login",
		HomeURL:  "https://baijiahao.baidu.com",
		Category: "social_media",
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

func GetSocialMediaPlatforms() []string {
	platforms := []string{}
	for key, config := range Platforms {
		if config.Category == "social_media" {
			platforms = append(platforms, key)
		}
	}
	return platforms
}

func IsSocialMediaPlatform(platform string) bool {
	return GetPlatformCategory(platform) == "social_media"
}
