package checkfile

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/md5"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

type EncryptedFile struct {
	OriginalPath  string // Початковий шлях
	EncryptedPath string // Шлях до зашифрованого файлу
	OriginalHash  string // MD5-хеш оригінального файлу
	EncryptedName string // Зашифрована назва (можна базувати на md5 або просто змінити розширення)
	OriginalSize  int64  // Розмір оригінального файлу
}

type FILEEncryptor struct {
	Key    []byte
	Input  <-chan string
	Output chan<- EncryptedFile
}

func NewFILEEncryptor(key []byte, input <-chan string, output chan<- EncryptedFile) (*FILEEncryptor, error) {
	if len(key) != 32 {
		return nil, fmt.Errorf("ключ повинен мати 32 байти для AES-256, отримано %d", len(key))
	}
	return &FILEEncryptor{Key: key, Input: input, Output: output}, nil
}

func (f *FILEEncryptor) Run() {
	block, err := aes.NewCipher(f.Key)
	if err != nil {
		panic(fmt.Sprintf("не вдалося ініціалізувати AES: %v", err))
	}

	for path := range f.Input {
		file, err := os.Open(path)
		if err != nil {
			fmt.Printf("не вдалося відкрити файл: %s\n", err)
			continue
		}

		stat, err := file.Stat()
		if err != nil {
			file.Close()
			fmt.Printf("не вдалося отримати інформацію про файл: %s\n", err)
			continue
		}
		size := stat.Size()

		hash := md5.New()
		_, err = io.Copy(hash, file)
		if err != nil {
			file.Close()
			fmt.Printf("не вдалося прочитати файл для хешу: %s\n", err)
			continue
		}
		hashStr := hex.EncodeToString(hash.Sum(nil))

		file.Seek(0, 0)
		iv := make([]byte, aes.BlockSize)
		if _, err := io.ReadFull(rand.Reader, iv); err != nil {
			file.Close()
			fmt.Printf("помилка генерації IV: %s\n", err)
			continue
		}

		stream := cipher.NewCFBEncrypter(block, iv)
		encryptedPath := path + ".enc"
		encryptedFile, err := os.Create(encryptedPath)
		if err != nil {
			file.Close()
			fmt.Printf("не вдалося створити зашифрований файл: %s\n", err)
			continue
		}

		if _, err = encryptedFile.Write(iv); err != nil {
			file.Close()
			encryptedFile.Close()
			fmt.Printf("не вдалося записати IV: %s\n", err)
			continue
		}

		writer := &cipher.StreamWriter{S: stream, W: encryptedFile}
		if _, err := io.Copy(writer, file); err != nil {
			fmt.Printf("не вдалося зашифрувати файл: %s\n", err)
		}

		file.Close()
		encryptedFile.Close()

		encFileName := filepath.Base(encryptedPath)

		f.Output <- EncryptedFile{
			OriginalPath:  path,
			EncryptedPath: encryptedPath,
			OriginalHash:  hashStr,
			EncryptedName: encFileName,
			OriginalSize:  size,
		}
	}
}
