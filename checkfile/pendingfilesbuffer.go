package checkfile

import (
	"encoding/json"
	"os"
	"sync"
)

type PendingFilesBuffer struct {
	mu     sync.RWMutex
	buffer map[string]Verify
}

// LoadFromFile завантажує дані з JSON-файлу
func (p *PendingFilesBuffer) LoadFromFile(path string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	file, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			p.buffer = make(map[string]Verify)
			return nil
		}
		return err
	}
	defer file.Close()

	var list []Verify
	if err := json.NewDecoder(file).Decode(&list); err != nil {
		return err
	}

	p.buffer = make(map[string]Verify)
	for _, v := range list {
		p.buffer[v.Path] = v
	}
	return nil
}

// AddToBuffer додає новий файл у буфер
func (p *PendingFilesBuffer) AddToBuffer(file Verify) {
	p.mu.Lock()
	defer p.mu.Unlock()

	// Перевіряємо, чи файл уже є в буфері
	existing, exists := p.buffer[file.Path] // повертає Verify

	if exists && existing.Hash == file.Hash {
		// Якщо хеш збігається, файл уже є і не потребує оновлення
		return
	}

	// Якщо файл новий або хеш змінився, додаємо або оновлюємо запис
	p.buffer[file.Path] = file
}

// SaveToFile зберігає буфер у JSON-файл
func (p *PendingFilesBuffer) SaveToFile(path string) error {
	p.mu.RLock()
	defer p.mu.RUnlock()

	var list []Verify
	for _, v := range p.buffer {
		list = append(list, v)
	}

	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	return json.NewEncoder(file).Encode(list)
}

// RemoveFromBuffer видаляє файл із буфера
func (p *PendingFilesBuffer) RemoveFromBuffer(filePath string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	delete(p.buffer, filePath)
}
