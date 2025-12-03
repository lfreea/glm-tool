package config

import (
	"os"
	"strconv"

	"github.com/joho/godotenv"
	"github.com/gophertool/tool/log"
)

type Config struct {
	Port            string
	TargetAPIURL    string
	AnthropicAPIURL string
	LogLevel        string
	Debug           bool
	DebugLogFile    string
	CachePath       string
	CacheTTLHours   int
}

var AppConfig *Config

func LoadConfig() {
	// 加载 .env 文件
	if err := godotenv.Load(); err != nil {
		log.Info("未找到 .env 文件，使用默认配置或环境变量")
	}

	AppConfig = &Config{
		Port:            getEnv("PORT", "8080"),
		TargetAPIURL:    getEnv("TARGET_API_URL", "https://open.bigmodel.cn/api/coding/paas/v4"),
		AnthropicAPIURL: getEnv("ANTHROPIC_API_URL", "https://open.bigmodel.cn/api/anthropic"),
		LogLevel:        getEnv("LOG_LEVEL", "info"),
		Debug:           getBoolEnv("DEBUG", false),
		DebugLogFile:    getEnv("DEBUG_LOG_FILE", "debug.json"),
		CachePath:       getEnv("CACHE_PATH", "image_cache.db"),
		CacheTTLHours:   getIntEnv("CACHE_TTL_HOURS", 24),
	}

	// 设置日志级别
	setLogLevel(AppConfig.LogLevel)

	log.Infof("配置加载完成: Port=%s, TargetAPIURL=%s, AnthropicAPIURL=%s, Debug=%v, LogLevel=%s",
		AppConfig.Port, AppConfig.TargetAPIURL, AppConfig.AnthropicAPIURL, AppConfig.Debug, AppConfig.LogLevel)
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getBoolEnv(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if b, err := strconv.ParseBool(value); err == nil {
			return b
		}
	}
	return defaultValue
}

func getIntEnv(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if i, err := strconv.Atoi(value); err == nil {
			return i
		}
	}
	return defaultValue
}

// setLogLevel 根据字符串设置日志级别
func setLogLevel(level string) {
	switch level {
	case "debug":
		log.SetLevel(log.DEBUG)
	case "info":
		log.SetLevel(log.INFO)
	case "warn", "warning":
		log.SetLevel(log.WARN)
	case "error":
		log.SetLevel(log.ERROR)
	case "data":
		log.SetLevel(log.DATA)
	case "none":
		log.SetLevel(log.NONE)
	default:
		log.SetLevel(log.INFO)
		log.Warnf("未知的日志级别: %s, 使用默认级别 INFO", level)
	}
}
