package configs

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// AES加密相关
var defaultKey []byte

// 密钥文件路径
var keyFilePath = filepath.Join(os.Getenv("HOME"), ".config", "zhipin-mcp", ".key")

func init() {
	// 从环境变量或密钥文件读取密钥
	key := os.Getenv("ZHIPIN_AES_KEY")
	if key == "" {
		// 尝试从密钥文件读取
		key = loadKeyFromFile()
	}
	if key == "" {
		// 生成随机密钥
		key = generateRandomKey()
		saveKeyToFile(key)
	}
	defaultKey = []byte(key)
}

// loadKeyFromFile 从文件加载密钥
func loadKeyFromFile() string {
	data, err := os.ReadFile(keyFilePath)
	if err != nil {
		return ""
	}
	return string(data)
}

// saveKeyToFile 保存密钥到文件
func saveKeyToFile(key string) {
	dir := filepath.Dir(keyFilePath)
	if err := os.MkdirAll(dir, 0700); err != nil {
		fmt.Printf("创建密钥目录失败: %v\n", err)
		return
	}
	// 保存密钥到文件，权限设为 0600
	if err := os.WriteFile(keyFilePath, []byte(key), 0600); err != nil {
		fmt.Printf("保存密钥文件失败: %v\n", err)
	}
}

// getDeviceKey 获取设备绑定密钥
func getDeviceKey() []byte {
	// 使用机器信息生成设备密钥
	host, _ := os.Hostname()
	user, _ := os.UserHomeDir()
	deviceInfo := host + user
	hash := sha256.Sum256([]byte(deviceInfo))
	return hash[:]
}

// generateRandomKey 生成32字节随机密钥
func generateRandomKey() string {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		panic("生成随机密钥失败")
	}
	return hex.EncodeToString(bytes)[:32]
}

// Encrypt 使用AES-256-GCM加密
func Encrypt(plaintext string) (string, error) {
	if plaintext == "" {
		return "", nil
	}

	block, err := aes.NewCipher(defaultKey)
	if err != nil {
		return "", fmt.Errorf("创建加密密钥失败: %v", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("创建GCM加密器失败: %v", err)
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("生成随机数失败: %v", err)
	}

	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// Decrypt 使用AES-256-GCM解密
func Decrypt(ciphertext string) (string, error) {
	if ciphertext == "" {
		return "", nil
	}

	data, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return "", fmt.Errorf("Base64解码失败: %v", err)
	}

	block, err := aes.NewCipher(defaultKey)
	if err != nil {
		return "", fmt.Errorf("创建解密密钥失败: %v", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("创建GCM解密器失败: %v", err)
	}

	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return "", fmt.Errorf("密文长度过短")
	}

	nonce, ciphertextBytes := data[:nonceSize], data[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertextBytes, nil)
	if err != nil {
		return "", fmt.Errorf("解密失败: %v", err)
	}

	return string(plaintext), nil
}
