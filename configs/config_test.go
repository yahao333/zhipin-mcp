package configs

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
)

// TestConfigStruct 测试配置结构体
func TestConfigStruct(t *testing.T) {
	cfg := &Config{
		Server: ServerConfig{
			Port: "8080",
		},
		Browser: BrowserConfig{
			Headless:    true,
			Bin:         "/usr/bin/chromium",
			UserDataDir: "/tmp/user-data",
		},
		Delivery: DeliveryConfig{
			MaxDaily:       50,
			MinDelay:       3000,
			MaxDelay:       8000,
			CheckDuplicate: true,
		},
		Account: AccountConfig{
			Username: "testuser",
			Password: "encrypted",
		},
		Cron: CronConfig{
			Enabled: true,
		},
		Log: LogConfig{
			Level: "info",
		},
		Database: DatabaseConfig{
			Path: "./data/test.db",
		},
	}

	assert.Equal(t, "8080", cfg.Server.Port)
	assert.True(t, cfg.Browser.Headless)
	assert.Equal(t, 50, cfg.Delivery.MaxDaily)
	assert.Equal(t, "testuser", cfg.Account.Username)
	assert.True(t, cfg.Cron.Enabled)
	assert.Equal(t, "info", cfg.Log.Level)
}

// TestServerConfigFields 测试服务器配置字段
func TestServerConfigFields(t *testing.T) {
	cfg := ServerConfig{
		Port: ":8080",
	}

	assert.Equal(t, ":8080", cfg.Port)
}

// TestBrowserConfigFields 测试浏览器配置字段
func TestBrowserConfigFields(t *testing.T) {
	cfg := BrowserConfig{
		Headless:    false,
		Bin:         "/path/to/bin",
		UserDataDir: "/path/to/data",
	}

	assert.False(t, cfg.Headless)
	assert.Equal(t, "/path/to/bin", cfg.Bin)
	assert.Equal(t, "/path/to/data", cfg.UserDataDir)
}

// TestDeliveryConfigFields 测试投递配置字段
func TestDeliveryConfigFields(t *testing.T) {
	cfg := DeliveryConfig{
		MaxDaily:       100,
		MinDelay:       1000,
		MaxDelay:       5000,
		CheckDuplicate: false,
	}

	assert.Equal(t, 100, cfg.MaxDaily)
	assert.Equal(t, 1000, cfg.MinDelay)
	assert.Equal(t, 5000, cfg.MaxDelay)
	assert.False(t, cfg.CheckDuplicate)
}

// TestAccountConfigFields 测试账号配置字段
func TestAccountConfigFields(t *testing.T) {
	cfg := AccountConfig{
		Username: "myuser",
		Password: "encrypted_password",
	}

	assert.Equal(t, "myuser", cfg.Username)
	assert.Equal(t, "encrypted_password", cfg.Password)
}

// TestCronConfigFields 测试定时任务配置字段
func TestCronConfigFields(t *testing.T) {
	cfg := CronConfig{
		Enabled: true,
	}

	assert.True(t, cfg.Enabled)
}

// TestLogConfigFields 测试日志配置字段
func TestLogConfigFields(t *testing.T) {
	tests := []struct {
		name  string
		level string
	}{
		{"Debug级别", "debug"},
		{"Info级别", "info"},
		{"Warn级别", "warn"},
		{"Error级别", "error"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := LogConfig{
				Level: tt.level,
			}
			assert.Equal(t, tt.level, cfg.Level)
		})
	}
}

// TestDatabaseConfigFields 测试数据库配置字段
func TestDatabaseConfigFields(t *testing.T) {
	cfg := DatabaseConfig{
		Path: "./data/zhipin.db",
	}

	assert.Equal(t, "./data/zhipin.db", cfg.Path)
}

// TestConfigYAML 测试 YAML 序列化
func TestConfigYAML(t *testing.T) {
	cfg := Config{
		Server: ServerConfig{
			Port: "8080",
		},
		Delivery: DeliveryConfig{
			MaxDaily: 50,
		},
	}

	// 测试序列化
	data, err := yaml.Marshal(cfg)
	assert.NoError(t, err)
	assert.Contains(t, string(data), "server:")
	assert.Contains(t, string(data), "delivery:")

	// 测试反序列化
	var parsed Config
	err = yaml.Unmarshal(data, &parsed)
	assert.NoError(t, err)
	assert.Equal(t, "8080", parsed.Server.Port)
	assert.Equal(t, 50, parsed.Delivery.MaxDaily)
}
