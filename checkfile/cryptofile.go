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
	"sync"
)

// FILEEncryptor — клас для шифрування файлів із Verify
type FILEEncryptor struct {
	Key    []byte
	Input  <-chan Verify
	Output chan<- EncryptedFile
	wg     *sync.WaitGroup
}

// NewFILEEncryptor — конструктор, перевіряє довжину ключа
func NewFILEEncryptor(key []byte, input <-chan Verify, output chan<- EncryptedFile) (*FILEEncryptor, error) {
	if len(key) != 32 {
		return nil, fmt.Errorf("ключ повинен мати 32 байти для AES-256, отримано %d", len(key))
	}
	return &FILEEncryptor{
		Key:    key,
		Input:  input,
		Output: output,
		wg:     nil, // буде встановлено через Start()
	}, nil
}

// Start — запускає Run() в окремій горутині
func (f *FILEEncryptor) Start(wg *sync.WaitGroup) {
	f.wg = wg
	wg.Add(1)
	go func() {
		defer wg.Done()
		f.Run()
	}()
}

// Run — основний цикл шифрування
func (f *FILEEncryptor) Run() {
	block, err := aes.NewCipher(f.Key)
	if err != nil {
		panic(fmt.Sprintf("не вдалося ініціалізувати AES: %v", err))
	}

	for verify := range f.Input {
		path := verify.Path

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

		// Обчислюємо MD5-хеш
		hash := md5.New()
		if _, err = io.Copy(hash, file); err != nil {
			file.Close()
			fmt.Printf("не вдалося прочитати файл для хешу: %s\n", err)
			continue
		}
		hashStr := hex.EncodeToString(hash.Sum(nil))

		if _, err := file.Seek(0, 0); err != nil {
			file.Close()
			fmt.Printf("не вдалося перемотати файл: %s\n", err)
			continue
		}

		iv := make([]byte, aes.BlockSize)
		if _, err := io.ReadFull(rand.Reader, iv); err != nil {
			file.Close()
			fmt.Printf("помилка генерації IV: %s\n", err)
			continue
		}

		stream := cipher.NewCFBEncrypter(block, iv)

		encryptedPath := path + ".enc"
		_ = os.Remove(encryptedPath)

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
			file.Close()
			encryptedFile.Close()
			fmt.Printf("не вдалося зашифрувати файл: %s\n", err)
			continue
		}

		file.Close()
		encryptedFile.Close()

		f.Output <- EncryptedFile{
			OriginalPath:  path,
			OriginalName:  filepath.Base(path),
			EncryptedPath: encryptedPath,
			OriginalHash:  hashStr,
			EncryptedName: filepath.Base(encryptedPath),
			OriginalSize:  size,
		}
	}
}
