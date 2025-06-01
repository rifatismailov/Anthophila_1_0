/*
+----------------------------+
|        FileExist           |
|                            |
| 1.  FilePathExists          |
|    +-------------------+   |
|    |                   |   |
|    | Читання JSON файлу|   |
|    | Перевірка та      |   |
|    | видалення шляху   |   |
|    +--------+----------+   |
|             |              |
|             v              |
| 2.  Запис оновленого        |
|     JSON файлу              |
|                            |
+----------------------------+
*/

package information

import (
	"encoding/json"
	"io"
	"os"
)

// FileExist представляє структуру для збереження шляхів до файлів.
type FileExist struct {
	FilePaths []string `json:"filePaths"` // Масив шляхів до файлів
}

// NewFileExist створює новий екземпляр FileExist.
func NewFileExist() *FileExist {
	return &FileExist{}
}

// FilePathExists перевіряє наявність вказаного шляху до файлу в JSON файлі.
// Якщо шлях знайдено, він видаляється з JSON файлу і файл перезаписується.
func (fs *FileExist) FilePathExists(filePath string, jsonFilePath string) (bool, error) {
	// Читання JSON-файлу
	file, err := os.Open(jsonFilePath)
	if err != nil {
		return false, err
	}
	defer file.Close()

	// Декодування JSON
	var store FileExist
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&store); err != nil && err != io.EOF {
		return false, err
	}

	// Перевірка наявності посилання
	found := false
	for i, path := range store.FilePaths {
		if path == filePath {
			// Видалення посилання
			store.FilePaths = append(store.FilePaths[:i], store.FilePaths[i+1:]...)
			found = true
			break
		}
	}

	if found {
		// Перезапис JSON-файлу
		file, err = os.Create(jsonFilePath)
		if err != nil {
			return false, err
		}
		defer file.Close()

		encoder := json.NewEncoder(file)
		encoder.SetIndent("", "  ")
		err = encoder.Encode(&store)
		if err != nil {
			return false, err
		}
	}

	return found, nil
}
