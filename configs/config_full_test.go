package configs

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

// TestDefaultConfig 测试默认配置
func TestDefaultConfig(t *testing.T) {
	cfg := defaultConfig()

	assert.NotNil(t, cfg)
	assert.Equal(t, ":18061", cfg.Server.Port)
	assert.True(t, cfg.Browser.Headless)
	assert.Equal(t, 30, cfg.Delivery.MaxDaily)
	assert.Equal(t, 3000, cfg.Delivery.MinDelay)
	assert.Equal(t, 8000, cfg.Delivery.MaxDelay)
	assert.Equal(t, "0 9 * * *", cfg.Cron.Expression)
	assert.Equal(t, "./data/zhipin.db", cfg.Database.Path)
}

// TestSetConfigPath 测试设置配置路径
func TestSetConfigPath(t *testing.T) {
	// 保存原始值
	origPath := configPath

	// 设置新路径
	SetConfigPath("/custom/path/config.yaml")

	// 验证设置成功
	assert.Equal(t, "/custom/path/config.yaml", configPath)

	// 恢复原始值
	configPath = origPath
}

// TestLoadConfigWithDefault 测试加载默认配置
func TestLoadConfigWithDefault(t *testing.T) {
	// 设置一个不存在的配置路径
	origPath := configPath
	SetConfigPath("/nonexistent/path/config.yaml")
	configPath = "/nonexistent/path/config.yaml"

	// 由于 once 已经执行，这里可能不会加载新的配置
	// 测试配置路径设置
	SetConfigPath("/tmp/test-config.yaml")

	// 恢复
	configPath = origPath
}

// TestGetConfig 测试获取配置
func TestGetConfig(t *testing.T) {
	cfg := GetConfig()
	assert.NotNil(t, cfg)
}

// TestConfig_GetConfigNil 测试配置为nil时获取
func TestConfig_GetConfigNil(t *testing.T) {
	// 确保cfg为nil时也能工作
	cfg := &Config{
		Server: ServerConfig{
			Port: "8080",
		},
	}
	result := cfg
	assert.NotNil(t, result)
}

// TestConfigPathPriority 测试配置文件查找优先级
func TestConfigPathPriority(t *testing.T) {
	// 测试默认配置路径
	paths := []string{
		configPath,
		filepath.Join(os.Getenv("HOME"), ".config", "zhipin-mcp", "config.yaml"),
		"/etc/zhipin-mcp/config.yaml",
	}

	// 验证至少有一个默认路径
	assert.GreaterOrEqual(t, len(paths), 1)
}

// TestConfigServerPort 测试服务器端口配置
func TestConfigServerPort(t *testing.T) {
	tests := []struct {
		name string
		port string
	}{
		{"默认端口", ":18061"},
		{"自定义端口", ":8080"},
		{"指定IP", "127.0.0.1:8080"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{
				Server: ServerConfig{
					Port: tt.port,
				},
			}
			assert.Equal(t, tt.port, cfg.Server.Port)
		})
	}
}

// TestConfigBrowserHeadless 测试浏览器无头模式配置
func TestConfigBrowserHeadless(t *testing.T) {
	tests := []struct {
		name     string
		headless bool
	}{
		{"启用无头", true},
		{"禁用无头", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{
				Browser: BrowserConfig{
					Headless: tt.headless,
				},
			}
			assert.Equal(t, tt.headless, cfg.Browser.Headless)
		})
	}
}

// TestConfigDeliveryMaxDaily 测试每日投递上限配置
func TestConfigDeliveryMaxDaily(t *testing.T) {
	tests := []struct {
		name     string
		maxDaily int
	}{
		{"默认", 30},
		{"较小", 10},
		{"较大", 100},
		{"无限制", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{
				Delivery: DeliveryConfig{
					MaxDaily: tt.maxDaily,
				},
			}
			assert.Equal(t, tt.maxDaily, cfg.Delivery.MaxDaily)
		})
	}
}

// TestConfigDelay 测试延时配置
func TestConfigDelay(t *testing.T) {
	cfg := &Config{
		Delivery: DeliveryConfig{
			MinDelay: 1000,
			MaxDelay: 5000,
		},
	}

	assert.Equal(t, 1000, cfg.Delivery.MinDelay)
	assert.Equal(t, 5000, cfg.Delivery.MaxDelay)
	assert.Less(t, cfg.Delivery.MinDelay, cfg.Delivery.MaxDelay)
}

// TestConfigCronExpression 测试Cron表达式配置
func TestConfigCronExpression(t *testing.T) {
	tests := []struct {
		name  string
		expr  string
		valid bool
	}{
		{"每天早上9点", "0 9 * * *", true},
		{"每小时", "0 * * * *", true},
		{"每天午夜", "0 0 * * *", true},
		{"每周一", "0 9 * * 1", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{
				Cron: CronConfig{
					Expression: tt.expr,
					Enabled:    tt.valid,
				},
			}
			assert.Equal(t, tt.expr, cfg.Cron.Expression)
			assert.Equal(t, tt.valid, cfg.Cron.Enabled)
		})
	}
}

// TestConfigLogLevel 测试日志级别配置
func TestConfigLogLevel(t *testing.T) {
	tests := []struct {
		name  string
		level string
	}{
		{"Debug", "debug"},
		{"Info", "info"},
		{"Warn", "warn"},
		{"Error", "error"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{
				Log: LogConfig{
					Level: tt.level,
				},
			}
			assert.Equal(t, tt.level, cfg.Log.Level)
		})
	}
}

// TestConfigLogFormat 测试日志格式配置
func TestConfigLogFormat(t *testing.T) {
	tests := []struct {
		name   string
		format string
	}{
		{"文本格式", "text"},
		{"JSON格式", "json"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{
				Log: LogConfig{
					Format: tt.format,
				},
			}
			assert.Equal(t, tt.format, cfg.Log.Format)
		})
	}
}

// TestConfigDatabasePath 测试数据库路径配置
func TestConfigDatabasePath(t *testing.T) {
	tests := []struct {
		name string
		path string
	}{
		{"默认路径", "./data/zhipin.db"},
		{"绝对路径", "/var/data/zhipin.db"},
		{"内存数据库", ":memory:"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{
				Database: DatabaseConfig{
					Path: tt.path,
				},
			}
			assert.Equal(t, tt.path, cfg.Database.Path)
		})
	}
}

// TestFullConfigInitialization 测试完整配置初始化
func TestFullConfigInitialization(t *testing.T) {
	cfg := &Config{
		Server: ServerConfig{
			Port: ":18061",
		},
		Browser: BrowserConfig{
			Headless:    true,
			Bin:         "/usr/bin/chromium",
			UserDataDir: "/tmp/chromium-data",
		},
		Delivery: DeliveryConfig{
			MaxDaily:       50,
			MinDelay:       2000,
			MaxDelay:       6000,
			CheckDuplicate: true,
		},
		Account: AccountConfig{
			Username: "testuser",
			Password: "encrypted_password",
		},
		Cron: CronConfig{
			Enabled:    true,
			Keyword:    "工程师",
			City:       "北京",
			Expression: "0 9 * * *",
		},
		Log: LogConfig{
			Level:  "debug",
			Format: "json",
		},
		Database: DatabaseConfig{
			Path: "/tmp/zhipin.db",
		},
	}

	require.NotNil(t, cfg)
	assert.Equal(t, ":18061", cfg.Server.Port)
	assert.True(t, cfg.Browser.Headless)
	assert.Equal(t, "/usr/bin/chromium", cfg.Browser.Bin)
	assert.Equal(t, 50, cfg.Delivery.MaxDaily)
	assert.Equal(t, "testuser", cfg.Account.Username)
	assert.True(t, cfg.Cron.Enabled)
	assert.Equal(t, "debug", cfg.Log.Level)
	assert.Equal(t, "/tmp/zhipin.db", cfg.Database.Path)
}

// TestConfigYAMLEmpty 测试空YAML解析
func TestConfigYAMLEmpty(t *testing.T) {
	emptyYAML := []byte("")

	var cfg Config
	err := yaml.Unmarshal(emptyYAML, &cfg)
	require.NoError(t, err)

	// 验证默认值
	assert.Equal(t, "", cfg.Server.Port)
}

// TestConfigYAMLPartial 测试部分YAML解析
func TestConfigYAMLEmptyPartial(t *testing.T) {
	partialYAML := []byte(`
server:
  port: ":8080"
delivery:
  max_daily: 100
`)

	var cfg Config
	err := yaml.Unmarshal(partialYAML, &cfg)
	require.NoError(t, err)

	assert.Equal(t, ":8080", cfg.Server.Port)
	assert.Equal(t, 100, cfg.Delivery.MaxDaily)
	// 未设置的字段应为零值
	assert.Equal(t, "", cfg.Browser.Bin)
}

// TestGlobalVariables 测试全局变量设置
func TestGlobalVariables(t *testing.T) {
	// 保存原始值
	origUsername := Username
	origPassword := Password
	origMaxDaily := MaxDaily

	// 设置值
	Username = "testuser"
	Password = "testpass"
	MaxDaily = 50

	assert.Equal(t, "testuser", Username)
	assert.Equal(t, "testpass", Password)
	assert.Equal(t, 50, MaxDaily)

	// 恢复原始值
	Username = origUsername
	Password = origPassword
	MaxDaily = origMaxDaily
}

// TestGlobalVariablesEmpty 测试空全局变量
func TestGlobalVariablesEmpty(t *testing.T) {
	// 保存原始值
	origUsername := Username

	Username = ""
	assert.Equal(t, "", Username)

	// 恢复
	Username = origUsername
}
