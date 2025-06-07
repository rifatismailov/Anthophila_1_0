package checkfile

import (
	"encoding/json"
	"os"
	"sync"
)

// PendingFilesBuffer — буфер для зашифрованих файлів, які ще не були відправлені
type PendingFilesBuffer struct {
	mu     sync.RWMutex
	buffer map[string]EncryptedFile // ключ — EncryptedPath
}

// LoadFromFile завантажує дані з JSON-файлу
func (p *PendingFilesBuffer) LoadFromFile(path string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	file, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			p.buffer = make(map[string]EncryptedFile)
			return nil
		}
		return err
	}
	defer file.Close()

	var list []EncryptedFile
	if err := json.NewDecoder(file).Decode(&list); err != nil {
		return err
	}

	p.buffer = make(map[string]EncryptedFile)
	for _, v := range list {
		p.buffer[v.EncryptedPath] = v
	}
	return nil
}

// AddToBuffer додає новий зашифрований файл у буфер
func (p *PendingFilesBuffer) AddToBuffer(file EncryptedFile) {
	p.mu.Lock()
	defer p.mu.Unlock()

	existing, exists := p.buffer[file.EncryptedPath]
	if exists && existing.OriginalHash == file.OriginalHash {
		// Якщо хеш збігається, нічого не робимо
		return
	}
	p.buffer[file.EncryptedPath] = file
}

// SaveToFile зберігає буфер у JSON-файл
func (p *PendingFilesBuffer) SaveToFile(path string) error {
	p.mu.RLock()
	defer p.mu.RUnlock()

	var list []EncryptedFile
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

// GetAllFiles повертає копію всіх файлів із буфера
func (p *PendingFilesBuffer) GetAllFiles() []EncryptedFile {
	p.mu.RLock()
	defer p.mu.RUnlock()

	var files []EncryptedFile
	for _, file := range p.buffer {
		files = append(files, file)
	}
	return files
}
