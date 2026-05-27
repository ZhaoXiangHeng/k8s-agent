// Package crypto 提供 LLM API Key 的 AES-256-GCM 加密/解密。
// 加密密钥从环境变量 ENCRYPTION_KEY 读取（开发模式无密钥时回退到明文存储）。
package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"io"
	"os"

	"github.com/sirupsen/logrus"
)

var pkgLog = logrus.WithField("component", "backend-api/crypto")

// getKey 从环境变量获取加密密钥，SHA-256 派生为 32 字节 AES-256 密钥。
// 开发模式（ENCRYPTION_KEY 为空）返回空密钥，使用明文存储。
func getKey() []byte {
	key := os.Getenv("ENCRYPTION_KEY")
	if key == "" {
		return nil
	}
	hash := sha256.Sum256([]byte(key))
	return hash[:]
}

// Encrypt 使用 AES-256-GCM 加密明文。
// 返回 base64 编码的密文（包含随机 nonce）。
// 开发模式下（未配置 ENCRYPTION_KEY）直接返回明文。
func Encrypt(plaintext string) (string, error) {
	if plaintext == "" {
		return "", nil
	}

	key := getKey()
	if key == nil {
		pkgLog.WithField("event", "encrypt_plaintext_fallback").Debug("no encryption key configured, storing as plaintext")
		return plaintext, nil
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("create cipher: %w", err)
	}

	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("create GCM: %w", err)
	}

	nonce := make([]byte, aesGCM.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("generate nonce: %w", err)
	}

	ciphertext := aesGCM.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// Decrypt 解密 base64 编码的 AES-256-GCM 密文。
// 开发模式下（未配置 ENCRYPTION_KEY）密文即为明文，直接返回。
func Decrypt(encoded string) (string, error) {
	if encoded == "" {
		return "", nil
	}

	key := getKey()
	if key == nil {
		return encoded, nil
	}

	ciphertext, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		// 可能是未加密的明文（从开发模式迁移到生产模式时）,直接返回
		return encoded, nil
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("create cipher: %w", err)
	}

	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("create GCM: %w", err)
	}

	nonceSize := aesGCM.NonceSize()
	if len(ciphertext) < nonceSize {
		// 密文太短，可能是明文，直接返回
		return encoded, nil
	}

	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	plaintext, err := aesGCM.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", fmt.Errorf("decrypt: %w", err)
	}

	return string(plaintext), nil
}
