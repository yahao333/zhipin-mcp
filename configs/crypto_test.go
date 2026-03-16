package configs

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestEncrypt 测试 AES 加密功能
func TestEncrypt(t *testing.T) {
	tests := []struct {
		name      string
		plaintext string
	}{
		{
			name:      "正常加密",
			plaintext: "Hello, World!",
		},
		{
			name:      "加密中文",
			plaintext: "你好，世界！",
		},
		{
			name:      "加密特殊字符",
			plaintext: "!@#$%^&*()_+-=[]{}|;':\",./<>?",
		},
		{
			name:      "加密长字符串",
			plaintext: "这是一段很长的测试文本，用于测试AES加密功能是否正常工作。Lorem ipsum dolor sit amet, consectetur adipiscing elit.",
		},
		{
			name:      "加密数字",
			plaintext: "1234567890",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 加密
			ciphertext, err := Encrypt(tt.plaintext)
			require.NoError(t, err, "加密失败")

			// 空字符串返回空
			if tt.plaintext == "" {
				assert.Equal(t, "", ciphertext)
				return
			}

			// 密文不应为空
			assert.NotEmpty(t, ciphertext, "密文不应为空")

			// 密文应与明文不同
			assert.NotEqual(t, tt.plaintext, ciphertext, "密文应与明文不同")
		})
	}
}

// TestDecrypt 测试 AES 解密功能
func TestDecrypt(t *testing.T) {
	tests := []struct {
		name      string
		plaintext string
	}{
		{
			name:      "正常解密",
			plaintext: "Hello, World!",
		},
		{
			name:      "解密中文",
			plaintext: "你好，世界！",
		},
		{
			name:      "解密特殊字符",
			plaintext: "!@#$%^&*()_+-=[]{}|;':\",./<>?",
		},
		{
			name:      "解密长字符串",
			plaintext: "这是一段很长的测试文本，用于测试AES加密功能是否正常工作。Lorem ipsum dolor sit amet, consectetur adipiscing elit.",
		},
		{
			name:      "解密数字",
			plaintext: "1234567890",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 先加密
			ciphertext, err := Encrypt(tt.plaintext)
			require.NoError(t, err, "加密失败")

			// 再解密
			decrypted, err := Decrypt(ciphertext)
			require.NoError(t, err, "解密失败")

			// 验证解密后的内容与原明文一致
			assert.Equal(t, tt.plaintext, decrypted, "解密后的内容应与原明文一致")
		})
	}
}

// TestEncryptEmpty 测试空字符串加密
func TestEncryptEmpty(t *testing.T) {
	result, err := Encrypt("")
	require.NoError(t, err)
	assert.Equal(t, "", result, "空字符串加密应返回空字符串")
}

// TestDecryptEmpty 测试空字符串解密
func TestDecryptEmpty(t *testing.T) {
	result, err := Decrypt("")
	require.NoError(t, err)
	assert.Equal(t, "", result, "空字符串解密应返回空字符串")
}

// TestDecryptInvalid 测试无效密文解密
func TestDecryptInvalid(t *testing.T) {
	tests := []struct {
		name        string
		ciphertext  string
		expectError bool
	}{
		{
			name:        "无效Base64",
			ciphertext:  "not-valid-base64!!!",
			expectError: true,
		},
		{
			name:        "过短密文",
			ciphertext:  "YWJj", // "abc" 的 Base64 编码
			expectError: true,
		},
		{
			name:        "无效密文",
			ciphertext:  "SGVsbG8gV29ybGQh", // "Hello World!" 的 Base64 但不是 AES-GCM 格式
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := Decrypt(tt.ciphertext)
			if tt.expectError {
				assert.Error(t, err, "应该返回错误")
			} else {
				require.NoError(t, err)
				assert.NotEmpty(t, result)
			}
			_ = result // 使用变量避免编译警告
		})
	}
}

// TestEncryptDecryptConsistency 测试加密解密一致性
func TestEncryptDecryptConsistency(t *testing.T) {
	// 多次加密同一明文应该产生不同的密文（因为随机 nonce）
	plaintext := "测试一致性"
	ciphertexts := make([]string, 5)

	for i := 0; i < 5; i++ {
		ct, err := Encrypt(plaintext)
		require.NoError(t, err)
		ciphertexts[i] = ct
	}

	// 验证所有密文都能正确解密
	for i, ct := range ciphertexts {
		decrypted, err := Decrypt(ct)
		require.NoError(t, err, "第%d次加密的密文解密失败", i+1)
		assert.Equal(t, plaintext, decrypted, "解密后应得到原始明文")
	}
}
