/*
+------------------------------------------------+
|                  Logging                       |
|                                                |
|   1.  Управління помилками файлів               |
|   +-----------------------------------------+  |
|   | LoadErrorPaths                          |  |
|   | SaveErrorPaths                          |  |
|   | IsPathInErrorList                       |  |
|   | AddErrorPath                            |  |
|   +-----------------------------------------+  |
|                                                |
+------------------------------------------------+
*/

package logging

import (
	"encoding/json"
	"os"
)

// ErrorPath представляє структуру для збереження шляху файлу та відповідної помилки.
type ErrorPath struct {
	Path  string `json:"path"`
	Error string `json:"error"`
}

// ErrorPaths представляє структуру для збереження списку помилкових шляхів.
type ErrorPaths struct {
	Paths []ErrorPath `json:"paths"`
}

const errorFilePath = "error_paths.json"

// LoadErrorPaths Завантаження помилок з JSON-файлу.
// Повертає список шляхів з помилками, або новий список, якщо файл не існує.
func LoadErrorPaths() (*ErrorPaths, error) {
	file, err := os.Open(errorFilePath)
	if err != nil {
		if os.IsNotExist(err) {
			return &ErrorPaths{}, nil
		}
		return nil, err
	}
	defer file.Close()

	var errorPaths ErrorPaths
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&errorPaths); err != nil {
		return nil, err
	}

	return &errorPaths, nil
}

// SaveErrorPaths Збереження помилок до JSON-файлу.
// Приймає структуру ErrorPaths і зберігає її до файлу `error_paths.json`.
func SaveErrorPaths(errorPaths *ErrorPaths) error {
	file, err := os.Create(errorFilePath)
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	return encoder.Encode(errorPaths)
}

// IsPathInErrorList Перевірка, чи містить список шляхів з помилками певний шлях.
// Приймає шлях та список ErrorPaths, повертає true, якщо шлях знайдений, інакше false.
func IsPathInErrorList(path string, errorPaths *ErrorPaths) bool {
	for _, ep := range errorPaths.Paths {
		if ep.Path == path {
			return true
		}
	}
	return false
}

// AddErrorPath Додавання нового шляху помилки до списку.
// Приймає шлях, повідомлення про помилку та список ErrorPaths, і додає нову помилку до списку.
func AddErrorPath(path, errorMsg string, errorPaths *ErrorPaths) {
	errorPaths.Paths = append(errorPaths.Paths, ErrorPath{
		Path:  path,
		Error: errorMsg,
	})
}

//	Опис:
//	ErrorPath структура:
//	Структура для збереження інформації про шлях файлу та відповідну помилку.

//	ErrorPaths структура:
//	Структура для збереження списку шляхів з помилками.

//	LoadErrorPaths:
//	Завантажує список шляхів з помилками з JSON-файлу. Якщо файл не існує, створює та повертає новий порожній список.

//	SaveErrorPaths:
//	Зберігає список шляхів з помилками до JSON-файлу.

//	IsPathInErrorList:
//	Перевіряє, чи міститься певний шлях у списку шляхів з помилками. Повертає true, якщо шлях знайдений, інакше false.

//	AddErrorPath:
//	Додає новий шлях та відповідну помилку до списку ErrorPaths.
