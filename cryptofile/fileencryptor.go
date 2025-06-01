package cryptofile

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"fmt"
	"io"
	"os"
)

type FILEEncryptor struct {
	Key []byte
}

// NewFILEEncryptor створює новий екземпляр з перевіркою ключа
func NewFILEEncryptor(key []byte) (*FILEEncryptor, error) {
	if len(key) != 32 {
		return nil, fmt.Errorf("ключ повинен мати рівно 32 байти для AES-256 (отримано %d)", len(key))
	}
	return &FILEEncryptor{Key: key}, nil
}

// EncryptingFile шифрує файл і зберігає .enc версію
func (f *FILEEncryptor) EncryptingFile(file *os.File) (*os.File, error) {
	encryptedFilePath := file.Name() + ".enc"

	encryptedFile, err := os.Create(encryptedFilePath)
	if err != nil {
		return nil, err
	}

	block, err := aes.NewCipher(f.Key)
	if err != nil {
		return nil, err
	}

	iv := make([]byte, aes.BlockSize)
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		return nil, err
	}

	if _, err = encryptedFile.Write(iv); err != nil {
		return nil, err
	}

	stream := cipher.NewCFBEncrypter(block, iv)
	writer := &cipher.StreamWriter{S: stream, W: encryptedFile}

	if _, err := io.Copy(writer, file); err != nil {
		return nil, err
	}

	encryptedFile.Seek(0, 0)
	return encryptedFile, nil
}
