package cookies

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
)

type Cookier interface {
	LoadCookies() ([]byte, error)
	SaveCookies(data []byte) error
	DeleteCookies() error
}

type localCookie struct {
	path string
}

func NewLoadCookie(path string) Cookier {
	if path == "" {
		panic("path is required")
	}

	return &localCookie{
		path: path,
	}
}

// LoadCookies 从文件中加载 cookies
func (c *localCookie) LoadCookies() ([]byte, error) {
	data, err := os.ReadFile(c.path)
	if err != nil {
		return nil, errors.Wrap(err, "读取cookies文件失败")
	}

	return data, nil
}

// SaveCookies 保存 cookies 到文件
func (c *localCookie) SaveCookies(data []byte) error {
	// 确保目录存在
	dir := filepath.Dir(c.path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return errors.Wrap(err, "创建目录失败")
	}

	return os.WriteFile(c.path, data, 0600)
}

// DeleteCookies 删除 cookies 文件
func (c *localCookie) DeleteCookies() error {
	if _, err := os.Stat(c.path); os.IsNotExist(err) {
		return nil
	}
	return os.Remove(c.path)
}

// GetCookiesFilePath 获取 cookies 文件路径
func GetCookiesFilePath() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		homeDir = "."
	}

	configDir := filepath.Join(homeDir, ".config", "zhipin-mcp")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return "cookies.json"
	}

	return filepath.Join(configDir, "cookies.json")
}

// IsCookieNotFound 检查错误是否是 cookies 文件不存在
func IsCookieNotFound(err error) bool {
	if err == nil {
		return false
	}
	return errors.Is(err, os.ErrNotExist) || strings.Contains(err.Error(), "no such file or directory")
}
