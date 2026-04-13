package config

import (
	"fmt"
	"os"
	"strconv"
)

// Config 保存整个应用运行所需的所有配置
type Config struct {
	// 服务配置
	ServerPort int

	// DeepSeek LLM 配置
	DeepSeekAPIKey  string
	DeepSeekBaseURL string
	DeepSeekModel   string

	// CSQAQ 数据源配置
	CSQAQToken   string
	CSQAQBaseURL string

	// 可选: Tavily 搜索 (情报研究员 Agent 会用)
	TavilyAPIKey string
	// 在 Config 结构体里加一个字段:
	TavilyBaseURL string
}

// Load 从环境变量加载配置，必填项缺失时返回错误
func Load() (*Config, error) {
	cfg := &Config{
		ServerPort:      getEnvInt("SERVER_PORT", 8080),
		DeepSeekAPIKey:  os.Getenv("DEEPSEEK_API_KEY"),
		DeepSeekBaseURL: getEnvStr("DEEPSEEK_BASE_URL", "https://api.deepseek.com"),
		DeepSeekModel:   getEnvStr("DEEPSEEK_MODEL", "deepseek-chat"),
		CSQAQToken:      os.Getenv("CSQAQ_TOKEN"),
		CSQAQBaseURL:    getEnvStr("CSQAQ_BASE_URL", "https://api.csqaq.com"),
		TavilyAPIKey:    os.Getenv("TAVILY_API_KEY"), // 可选
		TavilyBaseURL:   getEnvStr("TAVILY_BASE_URL", "https://api.tavily.com"),
	}

	// 必填项校验: 缺了就直接报错, 防止程序跑起来再崩
	if cfg.DeepSeekAPIKey == "" {
		return nil, fmt.Errorf("环境变量 DEEPSEEK_API_KEY 未设置")
	}
	if cfg.CSQAQToken == "" {
		return nil, fmt.Errorf("环境变量 CSQAQ_TOKEN 未设置")
	}

	return cfg, nil
}

// getEnvStr 读取字符串环境变量, 不存在则使用默认值
func getEnvStr(key, defaultVal string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultVal
}

// getEnvInt 读取整数环境变量, 不存在或解析失败则使用默认值
func getEnvInt(key string, defaultVal int) int {
	v := os.Getenv(key)
	if v == "" {
		return defaultVal
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return defaultVal
	}
	return n
}
