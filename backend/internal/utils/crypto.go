package utils

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"io"
	"os"
)

// 获取内部密钥 (可由系统环境变量提供，若无则使用固定安全派生密钥)
func getSecretKey() []byte {
	secret := os.Getenv("NVR_SECRET_KEY")
	if secret == "" {
		secret = "istore-nvr-default-secure-key-2026"
	}
	hash := sha256.Sum256([]byte(secret))
	return hash[:]
}

// EncryptPassword 使用 AES-GCM 加密敏感密码
func EncryptPassword(plainText string) (string, error) {
	if plainText == "" {
		return "", nil
	}
	key := getSecretKey()
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}

	ciphertext := gcm.Seal(nonce, nonce, []byte(plainText), nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// DecryptPassword 解密敏感密码
func DecryptPassword(cipherTextBase64 string) (string, error) {
	if cipherTextBase64 == "" {
		return "", nil
	}
	ciphertext, err := base64.StdEncoding.DecodeString(cipherTextBase64)
	if err != nil {
		return "", err
	}

	key := getSecretKey()
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return "", errors.New("密文数据长度不足")
	}

	nonce, actualCipher := ciphertext[:nonceSize], ciphertext[nonceSize:]
	plainText, err := gcm.Open(nil, nonce, actualCipher, nil)
	if err != nil {
		return "", err
	}

	return string(plainText), nil
}
