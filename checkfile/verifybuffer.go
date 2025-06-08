package checkfile

import (
	v "Anthophila/struct_modul"
	"crypto/sha256" // для обчислення SHA-256 хешу файлу
	"encoding/json" // для серіалізації/десеріалізації даних у JSON
	"fmt"           // для форматування рядків
	"io"            // для копіювання вмісту файлу у хешер
	"os"            // для роботи з файлами
	"path/filepath" // для виділення імені файлу з повного шляху
	"sync"          // для забезпечення потокобезпеки
)

///////////////////////////////////////////////////////////////////////////////
// Структура: VerifyBuffer
// Містить буфер перевірених файлів у вигляді мапи та забезпечує доступ до них
///////////////////////////////////////////////////////////////////////////////

type VerifyBuffer struct {
	mu     sync.RWMutex        // М’ютекс для потокобезпечного доступу до буфера
	buffer map[string]v.Verify // Основна мапа: ключ — шлях до файлу, значення — структура Verify
}

///////////////////////////////////////////////////////////////////////////////
// Метод: LoadFromFile
// Завантажує JSON-файл, який містить список перевірених файлів, у буфер (map)
// Якщо файл не існує — ініціалізує порожню мапу
///////////////////////////////////////////////////////////////////////////////

func (vb *VerifyBuffer) LoadFromFile(path string) error {
	vb.mu.Lock()         // Забороняємо іншим потокам змінювати мапу
	defer vb.mu.Unlock() // Розблокуємо після завершення

	file, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			vb.buffer = make(map[string]v.Verify) // Якщо файл відсутній — створюємо порожню мапу
			return nil
		}
		return err // Інші помилки повертаємо
	}
	defer file.Close()

	var list []v.Verify
	if err := json.NewDecoder(file).Decode(&list); err != nil {
		return err
	}

	vb.buffer = make(map[string]v.Verify)
	for _, v := range list {
		vb.buffer[v.Path] = v // Переносимо дані в мапу для швидкого доступу
	}
	return nil
}

///////////////////////////////////////////////////////////////////////////////
// Метод: SaveToBuffer
// Перевіряє чи файл змінився, і додає/оновлює запис у буфері
// Повертає true, якщо файл новий або змінений
///////////////////////////////////////////////////////////////////////////////

func (vb *VerifyBuffer) SaveToBuffer(filePath string) (bool, v.Verify, error) {
	hash, err := calculateHash(filePath) // Обчислюємо SHA-256 хеш
	if err != nil {
		return false, v.Verify{}, err
	}

	// Читаємо дані з буфера без блокування запису
	vb.mu.RLock()
	old, exists := vb.buffer[filePath]
	vb.mu.RUnlock()

	if exists && old.Hash == hash {
		return false, old, nil // Хеш не змінився — повертаємо існуючий об'єкт
	}

	newVerify := v.Verify{
		Path: filePath,
		Name: filepath.Base(filePath),
		Hash: hash,
	}

	// Запис змін — вимагає блокування
	vb.mu.Lock()
	vb.buffer[filePath] = newVerify
	vb.mu.Unlock()

	return true, newVerify, nil
}

///////////////////////////////////////////////////////////////////////////////
// Метод: SaveToFile
// Зберігає весь буфер у JSON-файл (перезаписує його)
///////////////////////////////////////////////////////////////////////////////

func (vb *VerifyBuffer) SaveToFile(path string) error {
	vb.mu.RLock() // Читання — достатньо RLock
	defer vb.mu.RUnlock()

	var list []v.Verify
	for _, v := range vb.buffer {
		list = append(list, v) // Перетворюємо map -> slice для збереження
	}

	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	return json.NewEncoder(file).Encode(list) // Записуємо slice у файл
}

///////////////////////////////////////////////////////////////////////////////
// Функція: calculateHash
// Обчислює SHA-256 хеш для переданого файлу
///////////////////////////////////////////////////////////////////////////////

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
