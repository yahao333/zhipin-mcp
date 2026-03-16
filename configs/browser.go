package configs

import (
	"os"
	"sync"
)

var (
	headless    = true
	binPath     = ""
	userDataDir = ""
	mu          sync.RWMutex
)

// InitHeadless 初始化无头模式设置
func InitHeadless(h bool) {
	mu.Lock()
	defer mu.Unlock()
	headless = h
}

// IsHeadless 返回是否无头模式
func IsHeadless() bool {
	mu.RLock()
	defer mu.RUnlock()
	return headless
}

// SetBinPath 设置浏览器二进制路径
func SetBinPath(path string) {
	mu.Lock()
	defer mu.Unlock()
	binPath = path
}

// GetBinPath 获取浏览器二进制路径
func GetBinPath() string {
	mu.RLock()
	defer mu.RUnlock()

	if binPath != "" {
		return binPath
	}

	// 从环境变量获取
	return os.Getenv("ROD_BROWSER_BIN")
}

// SetUserDataDir 设置用户数据目录
func SetUserDataDir(dir string) {
	mu.Lock()
	defer mu.Unlock()
	userDataDir = dir
}

// GetUserDataDir 获取用户数据目录
func GetUserDataDir() string {
	mu.RLock()
	defer mu.RUnlock()
	return userDataDir
}
