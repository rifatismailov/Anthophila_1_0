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

// FILEEncryptor — клас для шифрування файлів із Verify
type FILEEncryptor struct {
	Key    []byte               // Ключ AES-256 (32 байти)
	Input  <-chan Verify        // Канал із файлами на шифрування
	Output chan<- EncryptedFile // Канал із результатами шифрування
}

// NewFILEEncryptor — конструктор, перевіряє довжину ключа
func NewFILEEncryptor(key []byte, input <-chan Verify, output chan<- EncryptedFile) (*FILEEncryptor, error) {
	if len(key) != 32 {
		return nil, fmt.Errorf("ключ повинен мати 32 байти для AES-256, отримано %d", len(key))
	}
	return &FILEEncryptor{Key: key, Input: input, Output: output}, nil
}

// Run — основний цикл шифрування файлів із каналу Input
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

		// Повертаємо курсор у початок
		if _, err := file.Seek(0, 0); err != nil {
			file.Close()
			fmt.Printf("не вдалося перемотати файл: %s\n", err)
			continue
		}

		// Генеруємо IV
		iv := make([]byte, aes.BlockSize)
		if _, err := io.ReadFull(rand.Reader, iv); err != nil {
			file.Close()
			fmt.Printf("помилка генерації IV: %s\n", err)
			continue
		}

		stream := cipher.NewCFBEncrypter(block, iv)

		// Створюємо файл для зашифрованого вмісту
		encryptedPath := path + ".enc"
		_ = os.Remove(encryptedPath) // видаляємо стару версію, якщо існує

		encryptedFile, err := os.Create(encryptedPath)
		if err != nil {
			file.Close()
			fmt.Printf("не вдалося створити зашифрований файл: %s\n", err)
			continue
		}

		// Записуємо IV
		if _, err = encryptedFile.Write(iv); err != nil {
			file.Close()
			encryptedFile.Close()
			fmt.Printf("не вдалося записати IV: %s\n", err)
			continue
		}

		// Шифруємо файл
		writer := &cipher.StreamWriter{S: stream, W: encryptedFile}
		if _, err := io.Copy(writer, file); err != nil {
			file.Close()
			encryptedFile.Close()
			fmt.Printf("не вдалося зашифрувати файл: %s\n", err)
			continue
		}

		// Закриваємо файли
		file.Close()
		encryptedFile.Close()

		// Формуємо результат
		encFileName := filepath.Base(encryptedPath)

		f.Output <- EncryptedFile{
			OriginalPath:  path,
			OriginalName:  filepath.Base(path),
			EncryptedPath: encryptedPath,
			OriginalHash:  hashStr,
			EncryptedName: encFileName,
			OriginalSize:  size,
		}
	}
}
