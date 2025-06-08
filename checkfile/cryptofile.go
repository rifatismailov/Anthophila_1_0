///////////////////////////////////////////////////////////////////////////////
// Package: checkfile
// Клас: FILEEncryptor
// Опис:
//   Шифрує файли з типу Verify у формат AES-256 CFB.
//   Приймає файли через канал Input, обробляє, зберігає з розширенням ".enc"
//   і відправляє результат у канал Output як EncryptedFile.
///////////////////////////////////////////////////////////////////////////////

package checkfile

import (
	sm "Anthophila/struct_modul"
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

// /////////////////////////////////////////////////////////////////////////////
// Структура: FILEEncryptor
//
// Поля:
// - Key: 32-байтовий ключ для AES-256
// - Input: канал тільки для читання Verify, з якого надходять файли для шифрування
// - Output: канал тільки для запису EncryptedFile, в який надсилається результат
// - wg: вказівник на WaitGroup для контролю завершення горутини
// /////////////////////////////////////////////////////////////////////////////
type FILEEncryptor struct {
	Key    []byte                  // AES-256 ключ (обовʼязково 32 байти)
	Input  <-chan sm.Verify        // Канал для вхідних файлів
	Output chan<- sm.EncryptedFile // Канал для вихідних зашифрованих файлів
	wg     *sync.WaitGroup         // Синхронізація виконання (встановлюється в Start)
}

// /////////////////////////////////////////////////////////////////////////////
// Функція: NewFILEEncryptor
// Перевіряє довжину ключа і створює новий об'єкт FILEEncryptor.
//
// Повертає помилку, якщо ключ не 32 байти.
// /////////////////////////////////////////////////////////////////////////////
func NewFILEEncryptor(key []byte, input <-chan sm.Verify, output chan<- sm.EncryptedFile) (*FILEEncryptor, error) {
	if len(key) != 32 {
		return nil, fmt.Errorf("ключ повинен мати 32 байти для AES-256, отримано %d", len(key))
	}
	return &FILEEncryptor{
		Key:    key,
		Input:  input,
		Output: output,
		wg:     nil, // буде заданий у Start()
	}, nil
}

// /////////////////////////////////////////////////////////////////////////////
// Метод: Start
// Запускає горутину з методом Run() і додає її в WaitGroup.
// /////////////////////////////////////////////////////////////////////////////
func (f *FILEEncryptor) Start(wg *sync.WaitGroup) {
	f.wg = wg
	wg.Add(1)
	go func() {
		defer wg.Done()
		f.Run()
	}()
}

// /////////////////////////////////////////////////////////////////////////////
// Метод: Run
// Основний цикл шифрування файлів з Input-каналу.
// Створює IV, шифрує файл і записує в Output канал результат.
//
// Порядок дій:
// - читає файл
// - обчислює MD5-хеш оригінального файлу
// - генерує IV
// - шифрує потік AES-256 CFB
// - записує IV + зашифровані дані в новий файл
// /////////////////////////////////////////////////////////////////////////////
func (f *FILEEncryptor) Run() {
	// Ініціалізація AES блоку
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

		// Хешування (для перевірки унікальності)
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

		// Генерація IV (ініціалізаційного вектору)
		iv := make([]byte, aes.BlockSize)
		if _, err := io.ReadFull(rand.Reader, iv); err != nil {
			file.Close()
			fmt.Printf("помилка генерації IV: %s\n", err)
			continue
		}

		stream := cipher.NewCFBEncrypter(block, iv)

		// Створення нового шляху для зашифрованого файлу
		encryptedPath := path + ".enc"
		_ = os.Remove(encryptedPath)

		encryptedFile, err := os.Create(encryptedPath)
		if err != nil {
			file.Close()
			fmt.Printf("не вдалося створити зашифрований файл: %s\n", err)
			continue
		}

		// Запис IV на початок файлу
		if _, err = encryptedFile.Write(iv); err != nil {
			file.Close()
			encryptedFile.Close()
			fmt.Printf("не вдалося записати IV: %s\n", err)
			continue
		}

		// Створюємо потоковий writer з AES
		writer := &cipher.StreamWriter{S: stream, W: encryptedFile}
		if _, err := io.Copy(writer, file); err != nil {
			file.Close()
			encryptedFile.Close()
			fmt.Printf("не вдалося зашифрувати файл: %s\n", err)
			continue
		}

		file.Close()
		encryptedFile.Close()

		// Передаємо результат далі
		f.Output <- sm.EncryptedFile{
			OriginalPath:  path,
			OriginalName:  filepath.Base(path),
			EncryptedPath: encryptedPath,
			OriginalHash:  hashStr,
			EncryptedName: filepath.Base(encryptedPath),
			OriginalSize:  size,
		}
	}
}
