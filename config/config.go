package config

import (
	"log"
	"os"
)

// Config 应用配置结构
type Config struct {
	Port           string
	Production     bool
	AIEndpoint     string
	AIKey          string
	DeepSeekModel  string
	DeepSeekAPIKey string
	// AWS DynamoDB 配置
	AWSRegion      string
	DynamoDBTable  string
	EnableDynamoDB bool
}

// LoadConfig 加载应用配置
func LoadConfig() *Config {
	cfg := &Config{
		Port:           getEnv("PORT", "8080"),
		Production:     getEnv("GIN_MODE", "debug") == "release",
		AIEndpoint:     getEnv("AI_ENDPOINT", ""),
		AIKey:          getEnv("AI_KEY", ""),
		DeepSeekModel:  getEnv("DEEPSEEK_MODEL", "deepseek-chat"),
		DeepSeekAPIKey: getEnv("DEEPSEEK_API_KEY", "sk-e75601b8d3224e30aca1acf0b27964f8"),
		// AWS DynamoDB 配置
		AWSRegion:      getEnv("AWS_REGION", "ap-northeast-1"),
		DynamoDBTable:  getEnv("DYNAMODB_TABLE", "essay"),
		EnableDynamoDB: getEnv("ENABLE_DYNAMODB", "true") == "true",
	}

	// 如果未配置DeepSeek API密钥，输出警告
	if cfg.DeepSeekAPIKey == "" {
		log.Println("警告: 未配置DeepSeek API密钥，将使用模拟润色")
	}

	return cfg
}

// getEnv 获取环境变量，如果不存在则返回默认值
func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}
