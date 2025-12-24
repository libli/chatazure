package config

import (
	"os"

	"chatazure/tlog"

	"gopkg.in/yaml.v2"
)

const configFilename = "config.yaml"

// Config 配置信息
type Config struct {
	Web struct {
		Port   string `yaml:"port"`
		DBName string `yaml:"dbName"`
	} `yaml:"web"`
	Azure AzureConfig  `yaml:"azure"`
	Users []UserConfig `yaml:"users"`
}

// AzureConfig azure配置信息
type AzureConfig struct {
	Endpoint        string            `yaml:"endpoint"`
	ApiKey          string            `yaml:"api-key"`
	Deployments     map[string]string `yaml:"deployments"`
	WebSearchModels []string          `yaml:"web_search_models"` // 需要启用 web_search 的模型列表
}

// UserConfig 初始化用户信息
type UserConfig struct {
	Username string `yaml:"username"`
	Password string `yaml:"password"`
}

// Get 获取配置信息
func Get() (*Config, error) {
	data, err := os.ReadFile(configFilename)
	if err != nil {
		tlog.Error.Printf("failed to read config file: %v", err)
		return nil, err
	}
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		tlog.Error.Printf("failed to parse config file: %v", err)
		return nil, err
	}
	return &cfg, nil
}
