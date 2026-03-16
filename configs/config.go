package configs

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"gopkg.in/yaml.v3"
)

var (
	cfg        *Config
	once       sync.Once
	configPath = "config.yaml"
)

// Config 配置结构体
type Config struct {
	Server   ServerConfig   `yaml:"server"`
	Browser  BrowserConfig  `yaml:"browser"`
	Delivery DeliveryConfig `yaml:"delivery"`
	Account  AccountConfig  `yaml:"account"`
	Cron     CronConfig     `yaml:"cron"`
	Log      LogConfig      `yaml:"log"`
	Database DatabaseConfig `yaml:"database"`
}

// ServerConfig 服务配置
type ServerConfig struct {
	Port string `yaml:"port"`
}

// BrowserConfig 浏览器配置
type BrowserConfig struct {
	Headless    bool   `yaml:"headless"`
	Bin         string `yaml:"bin"`
	UserDataDir string `yaml:"user_data_dir"`
}

// DeliveryConfig 投递配置
type DeliveryConfig struct {
	MaxDaily       int  `yaml:"max_daily"`
	MinDelay       int  `yaml:"min_delay"`
	MaxDelay       int  `yaml:"max_delay"`
	CheckDuplicate bool `yaml:"check_duplicate"`
}

// AccountConfig 账号配置
type AccountConfig struct {
	Username string `yaml:"username"`
	Password string `yaml:"password"` // AES加密后的密码
}

// CronConfig 定时任务配置
type CronConfig struct {
	Enabled    bool   `yaml:"enabled"`
	Keyword    string `yaml:"keyword"`
	City       string `yaml:"city"`
	Expression string `yaml:"expression"`
}

// LogConfig 日志配置
type LogConfig struct {
	Level  string `yaml:"level"`
	Format string `yaml:"format"`
}

// DatabaseConfig 数据库配置
type DatabaseConfig struct {
	Path string `yaml:"path"`
}

// 全局变量
var (
	Username     string
	Password     string
	MaxDaily     int
	MinDelay     int
	MaxDelay     int
	Port         string
	DatabasePath string
)

// Init 初始化配置
func Init() error {
	var err error
	once.Do(func() {
		err = loadConfig()
	})
	return err
}

// loadConfig 加载配置文件
func loadConfig() error {
	// 查找配置文件
	paths := []string{
		configPath,
		filepath.Join(os.Getenv("HOME"), ".config", "zhipin-mcp", "config.yaml"),
		"/etc/zhipin-mcp/config.yaml",
	}

	var configFile string
	for _, p := range paths {
		if _, err := os.Stat(p); err == nil {
			configFile = p
			break
		}
	}

	if configFile == "" {
		// 使用默认配置
		cfg = defaultConfig()
		return nil
	}

	data, err := os.ReadFile(configFile)
	if err != nil {
		return fmt.Errorf("读取配置文件失败: %v", err)
	}

	cfg = &Config{}
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return fmt.Errorf("解析配置文件失败: %v", err)
	}

	// 设置全局变量
	Username = cfg.Account.Username
	Password = cfg.Account.Password
	MaxDaily = cfg.Delivery.MaxDaily
	MinDelay = cfg.Delivery.MinDelay
	MaxDelay = cfg.Delivery.MaxDelay
	Port = cfg.Server.Port
	DatabasePath = cfg.Database.Path

	return nil
}

// defaultConfig 返回默认配置
func defaultConfig() *Config {
	return &Config{
		Server: ServerConfig{
			Port: ":18061",
		},
		Browser: BrowserConfig{
			Headless: true,
		},
		Delivery: DeliveryConfig{
			MaxDaily:       30,
			MinDelay:       3000,
			MaxDelay:       8000,
			CheckDuplicate: true,
		},
		Cron: CronConfig{
			Enabled:    false,
			Expression: "0 9 * * *",
		},
		Log: LogConfig{
			Level:  "info",
			Format: "text",
		},
		Database: DatabaseConfig{
			Path: "./data/zhipin.db",
		},
	}
}

// GetConfig 获取配置
func GetConfig() *Config {
	if cfg == nil {
		loadConfig()
	}
	return cfg
}

// SetConfigPath 设置配置文件路径
func SetConfigPath(path string) {
	configPath = path
}
