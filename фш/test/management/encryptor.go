package management

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
)

type Encryptor struct {
	Key []byte
}

func NewEncryptor(key string) (*Encryptor, error) {
	fmt.Println("Створення нового Encryptor з ключем:", key)
	fmt.Println("Довжина ключа:", len(key))
	if len(key) != 32 {
		return nil, fmt.Errorf("ключ повинен мати рівно 32 символи (AES-256)")
	}
	return &Encryptor{Key: []byte(key)}, nil
}

// EncryptText шифрує текст і повертає Base64
func (e *Encryptor) EncryptText(plainText string) (string, error) {
	block, err := aes.NewCipher(e.Key)
	if err != nil {
		return "", err
	}

	plaintext := pad([]byte(plainText), aes.BlockSize)

	ciphertext := make([]byte, aes.BlockSize+len(plaintext))
	iv := ciphertext[:aes.BlockSize]

	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		return "", err
	}

	mode := cipher.NewCBCEncrypter(block, iv)
	mode.CryptBlocks(ciphertext[aes.BlockSize:], plaintext)

	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// DecryptText розшифровує текст з Base64
func (e *Encryptor) DecryptText(encryptedBase64 string) (string, error) {
	ciphertext, err := base64.StdEncoding.DecodeString(encryptedBase64)
	if err != nil {
		return "", err
	}

	block, err := aes.NewCipher(e.Key)
	if err != nil {
		return "", err
	}

	if len(ciphertext) < aes.BlockSize {
		return "", errors.New("ciphertext занадто короткий")
	}

	iv := ciphertext[:aes.BlockSize]
	ciphertext = ciphertext[aes.BlockSize:]

	mode := cipher.NewCBCDecrypter(block, iv)
	mode.CryptBlocks(ciphertext, ciphertext)

	plaintext := unpad(ciphertext)
	return string(plaintext), nil
}

// ======== Допоміжні функції ========

func pad(src []byte, blockSize int) []byte {
	padding := blockSize - len(src)%blockSize
	padtext := make([]byte, padding)
	for i := range padtext {
		padtext[i] = byte(padding)
	}
	return append(src, padtext...)
}

func unpad(src []byte) []byte {
	length := len(src)
	unpadding := int(src[length-1])
	if unpadding > length {
		return src
	}
	return src[:(length - unpadding)]
}
