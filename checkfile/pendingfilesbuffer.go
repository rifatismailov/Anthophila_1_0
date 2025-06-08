///////////////////////////////////////////////////////////////////////////////
// Package: checkfile
// Клас: PendingFilesBuffer
// Опис: Менеджер буфера зашифрованих файлів, які ще не були відправлені.
//       Забезпечує безпечну конкурентну роботу через RWMutex.
//       Підтримує збереження в файл, завантаження, додавання, видалення та перегляд.
///////////////////////////////////////////////////////////////////////////////

package checkfile

import (
	sm "Anthophila/struct_modul"
	"encoding/json"
	"os"
	"sync"
)

// /////////////////////////////////////////////////////////////////////////////
// Структура: PendingFilesBuffer
//
// Поля:
//   - mu: RWMutex для синхронізації читання/запису до буфера.
//   - buffer: мапа, де ключ — шлях до зашифрованого файлу (EncryptedPath),
//     а значення — структура EncryptedFile.
//
// Призначення:
// Цей буфер зберігає список файлів, які були зашифровані, але ще не відправлені.
// Використовується в горутинах для асинхронної роботи.
// /////////////////////////////////////////////////////////////////////////////
type PendingFilesBuffer struct {
	mu     sync.RWMutex                // М'ютекс для конкурентної синхронізації
	buffer map[string]sm.EncryptedFile // Мапа файлів, ключ — EncryptedPath
}

// /////////////////////////////////////////////////////////////////////////////
// Метод: LoadFromFile
// Завантажує буфер з JSON-файлу. Якщо файл не існує — створює порожній буфер.
//
// Параметри:
// - path: шлях до JSON-файлу.
//
// Повертає:
// - помилку, якщо вона виникла під час читання або декодування.
// /////////////////////////////////////////////////////////////////////////////
func (p *PendingFilesBuffer) LoadFromFile(path string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	file, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			p.buffer = make(map[string]sm.EncryptedFile) // Створити порожній буфер
			return nil
		}
		return err
	}
	defer file.Close()

	var list []sm.EncryptedFile
	if err := json.NewDecoder(file).Decode(&list); err != nil {
		return err
	}

	p.buffer = make(map[string]sm.EncryptedFile)
	for _, v := range list {
		p.buffer[v.EncryptedPath] = v
	}
	return nil
}

// /////////////////////////////////////////////////////////////////////////////
// Метод: AddToBuffer
// Додає зашифрований файл до буфера, якщо файл новий або хеш змінився.
//
// Параметри:
// - file: об'єкт EncryptedFile, який потрібно додати.
// /////////////////////////////////////////////////////////////////////////////
func (p *PendingFilesBuffer) AddToBuffer(file sm.EncryptedFile) {
	p.mu.Lock()
	defer p.mu.Unlock()

	existing, exists := p.buffer[file.EncryptedPath]
	if exists && existing.OriginalHash == file.OriginalHash {
		// 🟡 Файл уже в буфері і не змінився — нічого не робимо
		return
	}
	p.buffer[file.EncryptedPath] = file
}

// /////////////////////////////////////////////////////////////////////////////
// Метод: SaveToFile
// Зберігає весь буфер у JSON-файл (перезаписує його повністю).
//
// Параметри:
// - path: шлях до JSON-файлу.
//
// Повертає:
// - помилку, якщо вона виникла під час запису.
// /////////////////////////////////////////////////////////////////////////////
func (p *PendingFilesBuffer) SaveToFile(path string) error {
	p.mu.RLock() // Читання — достатньо RLock
	defer p.mu.RUnlock()

	var list []sm.EncryptedFile
	for _, v := range p.buffer {
		list = append(list, v)
	}

	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	return json.NewEncoder(file).Encode(list) // Записуємо slice у файл
}

// /////////////////////////////////////////////////////////////////////////////
// Метод: RemoveFromBuffer
// Видаляє файл з буфера за шляхом.
//
// Параметри:
// - filePath: шлях до зашифрованого файлу (EncryptedPath).
// /////////////////////////////////////////////////////////////////////////////
func (p *PendingFilesBuffer) RemoveFromBuffer(filePath string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	delete(p.buffer, filePath)
}

// /////////////////////////////////////////////////////////////////////////////
// Метод: GetAllFiles
// Повертає копію всіх файлів у вигляді slice.
//
// Повертає:
// - []EncryptedFile: список усіх файлів з буфера.
// /////////////////////////////////////////////////////////////////////////////
func (p *PendingFilesBuffer) GetAllFiles() []sm.EncryptedFile {
	p.mu.RLock()
	defer p.mu.RUnlock()

	var files []sm.EncryptedFile
	for _, file := range p.buffer {
		files = append(files, file)
	}
	return files
}
