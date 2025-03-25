package pkg

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"encoding/base64"
	"fmt"
) // encryptAESCBC 加密并返回 Base64 字符串

func EncryptAESCBC(plaintext, key []byte) (string, error) {
	key = adjustKeySize(key)
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("failed to create cipher: %v", err)
	}

	//默认iv为0
	iv := make([]byte, aes.BlockSize)
	if len(iv) != aes.BlockSize {
		iv = append(iv, make([]byte, aes.BlockSize-len(iv))...) //填充
	}
	// PKCS7 填充
	padding := aes.BlockSize - len(plaintext)%aes.BlockSize
	padtext := bytes.Repeat([]byte{byte(padding)}, padding)
	plaintext = append(plaintext, padtext...)

	// 加密
	mode := cipher.NewCBCEncrypter(block, iv)
	ciphertext := make([]byte, len(plaintext))
	mode.CryptBlocks(ciphertext, plaintext)

	// 将 IV 和密文拼接后转为 Base64 字符串
	fullCiphertext := append(iv, ciphertext...)
	return base64.StdEncoding.EncodeToString(fullCiphertext), nil
}

// DecryptAESCBC 从 Base64 字符串解密
func DecryptAESCBC(ciphertextBase64 string, key []byte) ([]byte, error) {
	ciphertext, err := base64.StdEncoding.DecodeString(ciphertextBase64)
	if err != nil {
		return nil, fmt.Errorf("failed to decode base64: %v", err)
	}
	key = adjustKeySize(key)
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %v", err)
	}
	if len(ciphertext) < aes.BlockSize {
		return nil, fmt.Errorf("ciphertext too short")
	}
	iv := ciphertext[:aes.BlockSize]
	ciphertext = ciphertext[aes.BlockSize:]

	if len(ciphertext)%aes.BlockSize != 0 {
		return nil, fmt.Errorf("ciphertext is not a multiple of block size")
	}
	mode := cipher.NewCBCDecrypter(block, iv)
	plaintext := make([]byte, len(ciphertext))
	mode.CryptBlocks(plaintext, ciphertext)

	padding := int(plaintext[len(plaintext)-1])
	if padding > len(plaintext) || padding > aes.BlockSize {
		return nil, fmt.Errorf("invalid padding")
	}
	return plaintext[:len(plaintext)-padding], nil
}

// adjustKeySize 调整密钥大小为 AES 所需的长度（16、24 或 32 字节）
func adjustKeySize(key []byte) []byte {
	keyLen := len(key)

	// 如果密钥长度已经是 16、24 或 32 字节，直接返回
	if keyLen == 16 || keyLen == 24 || keyLen == 32 {
		return key
	}

	// 根据原始密钥长度选择目标长度
	var targetSize int
	if keyLen < 16 {
		targetSize = 16 // AES-128
	} else if keyLen < 24 {
		targetSize = 24 // AES-192
	} else {
		targetSize = 32 // AES-256
	}
	// 创建新的密钥
	newKey := make([]byte, targetSize)
	// 如果原始密钥太长，截断它
	if keyLen > targetSize {
		copy(newKey, key[:targetSize])
	} else {
		// 如果原始密钥太短，复制它并重复填充
		copy(newKey, key)
		for i := keyLen; i < targetSize; i++ {
			newKey[i] = key[i%keyLen]
		}
	}
	return newKey
}
