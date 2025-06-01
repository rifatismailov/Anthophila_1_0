/*
+----------------------------+
|         FileInfo           |
|                            |
| 1.   CheckAndWriteHash      |
|    +-------------------+   |
|    |                   |   |
|    | Обчислення хешу  |   |
|    | та запис у JSON  |   |
|    +--------+----------+   |
|             |              |
|             v              |
| 2.   calculateHash         |
|     +-------------------+  |
|     |                   |  |
|     | Обчислення хешу  |  |
|     +-------------------+  |
|                            |
+----------------------------+
*/

package checkfile

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// FileInfo представляє інформацію про файл, включаючи шлях, ім'я та хеш.
type FileInfo struct {
	Path string `json:"path"` // Шлях до файлу
	Name string `json:"name"` // Ім'я файлу
	Hash string `json:"hash"` // Хеш файлу
}

// NewFileInfo створює новий екземпляр FileInfo.
func NewFileInfo() *FileInfo {
	return &FileInfo{}
}

// CheckAndWriteHash обчислює хеш для файлу, порівнює його з існуючими даними в JSON файлі,
// оновлює JSON файл, якщо хеш змінився, або додає новий запис, якщо файл не знайдено.
func (fi FileInfo) CheckAndWriteHash(filePath string, jsonFilePath string) (bool, error) {
	// Обчислення хешу файлу
	hash, err := calculateHash(filePath)
	if err != nil {
		return false, err
	}

	// Читання існуючих даних з JSON файлу
	var existingData []FileInfo
	if _, err := os.Stat(jsonFilePath); !os.IsNotExist(err) {
		file, err := os.Open(jsonFilePath)
		if err != nil {
			return false, err
		}
		defer file.Close()

		decoder := json.NewDecoder(file)
		err = decoder.Decode(&existingData)
		if err != nil && err != io.EOF {
			return false, err
		}
	}

	// Пошук інформації про файл в JSON
	for i, info := range existingData {
		if info.Path == filePath {
			// Хеш змінився, оновлюємо інформацію
			if info.Hash != hash {
				existingData[i].Hash = hash

				// Запис оновлених даних в JSON файл
				file, err := os.Create(jsonFilePath)
				if err != nil {
					return false, err
				}
				defer file.Close()

				encoder := json.NewEncoder(file)
				err = encoder.Encode(existingData)
				if err != nil {
					return false, err
				}

				return true, nil // Хеш змінився і був оновлений
			}
			return false, nil // Хеш не змінився
		}
	}

	// Файл не знайдено в JSON, додаємо новий запис
	existingData = append(existingData, FileInfo{
		Path: filePath,
		Name: filepath.Base(filePath),
		Hash: hash,
	})

	// Запис оновлених даних в JSON файл
	file, err := os.Create(jsonFilePath)
	if err != nil {
		return false, err
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	err = encoder.Encode(existingData)
	if err != nil {
		return false, err
	}

	return true, nil // Файл додано або оновлено
}

// calculateHash обчислює SHA-256 хеш файлу за вказаним шляхом.
func calculateHash(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", hash.Sum(nil)), nil
}

//	Опис:
//	FileInfo структура:
//	Path: Шлях до файлу.
//	Name: Ім'я файлу.
//	Hash: Хеш файлу.

//	NewFileInfo:
//	Функція для створення нового екземпляра FileInfo.

//	CheckAndWriteHash:
//	Обчислює хеш файлу.
//	Порівнює хеш з даними в JSON файлі.
//	Оновлює JSON файл, якщо хеш змінився, або додає новий запис, якщо файл не знайдено.

//	calculateHash:
//	Обчислює SHA-256 хеш файлу і повертає його як рядок.
