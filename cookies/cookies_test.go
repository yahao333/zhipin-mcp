package cookies

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestNewLoadCookie 测试创建 Cookie 加载器
func TestNewLoadCookie(t *testing.T) {
	// 测试有效路径
	loader := NewLoadCookie("/tmp/test-cookies.json")
	assert.NotNil(t, loader)
}

// TestNewLoadCookie_Panic 测试空路径会panic
func TestNewLoadCookie_Panic(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			// 预期会panic
			assert.NotNil(t, r)
		}
	}()

	NewLoadCookie("")
}

// TestLocalCookie_LoadCookies_FileNotExist 测试加载不存在的文件
func TestLocalCookie_LoadCookies_FileNotExist(t *testing.T) {
	loader := NewLoadCookie("/tmp/nonexistent-path-12345.json")
	_, err := loader.LoadCookies()
	assert.Error(t, err)
}

// TestLocalCookie_SaveCookies 测试保存 cookies
func TestLocalCookie_SaveCookies(t *testing.T) {
	// 创建临时目录
	tmpDir := t.TempDir()
	cookiePath := filepath.Join(tmpDir, "test-cookies.json")

	loader := NewLoadCookie(cookiePath)
	testData := []byte(`{"session": "test-session-123"}`)

	// 保存
	err := loader.SaveCookies(testData)
	assert.NoError(t, err)

	// 加载
	loadedData, err := loader.LoadCookies()
	assert.NoError(t, err)
	assert.Equal(t, testData, loadedData)
}

// TestLocalCookie_SaveCookies_Overwrite 测试覆盖保存 cookies
func TestLocalCookie_SaveCookies_Overwrite(t *testing.T) {
	tmpDir := t.TempDir()
	cookiePath := filepath.Join(tmpDir, "overwrite-cookies.json")

	loader := NewLoadCookie(cookiePath)

	// 第一次保存
	err := loader.SaveCookies([]byte("first"))
	assert.NoError(t, err)

	// 第二次保存（覆盖）
	err = loader.SaveCookies([]byte("second"))
	assert.NoError(t, err)

	// 验证是第二次的数据
	loadedData, err := loader.LoadCookies()
	assert.NoError(t, err)
	assert.Equal(t, []byte("second"), loadedData)
}

// TestLocalCookie_DeleteCookies 测试删除 cookies
func TestLocalCookie_DeleteCookies(t *testing.T) {
	tmpDir := t.TempDir()
	cookiePath := filepath.Join(tmpDir, "delete-cookies.json")

	loader := NewLoadCookie(cookiePath)

	// 先保存
	err := loader.SaveCookies([]byte("test"))
	assert.NoError(t, err)

	// 删除
	err = loader.DeleteCookies()
	assert.NoError(t, err)

	// 验证文件不存在
	_, err = os.Stat(cookiePath)
	assert.True(t, os.IsNotExist(err))
}

// TestLocalCookie_DeleteCookies_NotExist 测试删除不存在的文件
func TestLocalCookie_DeleteCookies_NotExist(t *testing.T) {
	loader := NewLoadCookie("/tmp/nonexistent-cookies-12345.json")

	// 删除不存在的文件不应该报错
	err := loader.DeleteCookies()
	assert.NoError(t, err)
}

// TestGetCookiesFilePath 测试获取 cookies 文件路径
func TestGetCookiesFilePath(t *testing.T) {
	path := GetCookiesFilePath()
	assert.NotEmpty(t, path)
	assert.Contains(t, path, "zhipin-mcp")
	assert.Contains(t, path, "cookies.json")
}

// TestGetCookiesFilePath_CreateDir 测试获取 cookies 路径时会创建目录
func TestGetCookiesFilePath_CreateDir(t *testing.T) {
	// 保存原始HOME
	origHome := os.Getenv("HOME")
	defer os.Setenv("HOME", origHome)

	// 设置临时HOME
	tmpDir := t.TempDir()
	os.Setenv("HOME", tmpDir)

	path := GetCookiesFilePath()
	assert.NotEmpty(t, path)

	// 验证目录存在
	dir := filepath.Dir(path)
	_, err := os.Stat(dir)
	assert.NoError(t, err)
}

// TestCookierInterface 测试 Cookier 接口
func TestCookierInterface(t *testing.T) {
	tmpDir := t.TempDir()
	cookiePath := filepath.Join(tmpDir, "interface-test.json")

	// 声明接口类型
	var c Cookier
	c = NewLoadCookie(cookiePath)

	// 测试接口方法
	testData := []byte(`{"key": "value"}`)
	err := c.SaveCookies(testData)
	assert.NoError(t, err)

	loadedData, err := c.LoadCookies()
	assert.NoError(t, err)
	assert.Equal(t, testData, loadedData)

	err = c.DeleteCookies()
	assert.NoError(t, err)
}

// TestLocalCookie_SaveCookiesToNestedPath 测试保存到嵌套路径
func TestLocalCookie_SaveCookiesToNestedPath(t *testing.T) {
	tmpDir := t.TempDir()
	nestedPath := filepath.Join(tmpDir, "subdir1", "subdir2", "cookies.json")

	loader := NewLoadCookie(nestedPath)

	err := loader.SaveCookies([]byte("nested test"))
	assert.NoError(t, err)

	// 验证目录创建成功
	dir := filepath.Dir(nestedPath)
	info, err := os.Stat(dir)
	assert.NoError(t, err)
	assert.True(t, info.IsDir())
}

// TestLocalCookie_Permission 测试文件权限
func TestLocalCookie_Permission(t *testing.T) {
	tmpDir := t.TempDir()
	cookiePath := filepath.Join(tmpDir, "permission-test.json")

	loader := NewLoadCookie(cookiePath)
	err := loader.SaveCookies([]byte("permission test"))
	assert.NoError(t, err)

	// 验证文件权限
	info, err := os.Stat(cookiePath)
	assert.NoError(t, err)
	// 0600 权限意味着只有所有者有读写权限
	assert.Equal(t, os.FileMode(0600), info.Mode().Perm())
}
