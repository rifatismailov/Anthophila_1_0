package information

import (
	"encoding/json"
	"io"
	"os"
)

// AddPath зберігає список шляхів до файлів у JSON-файлі.
type AddPath struct {
	// FilePaths містить список шляхів до файлів, які потрібно зберегти у JSON-файлі.
	FilePaths []string `json:"filePaths"`
}

// NewAddPath створює новий екземпляр AddPath.
func NewAddPath() *AddPath {
	return &AddPath{}
}

// AddFilePath додає новий шлях до файлу у JSON-файл.
//
// Аргументи:
// - filePath: шлях до файлу, який потрібно додати.
// - jsonFilePath: шлях до JSON-файлу, в якому зберігається список шляхів.
//
// Функція виконує такі дії:
// 1. Відкриває JSON-файл для читання та запису, створюючи його, якщо він не існує.
// 2. Декодує існуючий JSON у структуру AddPath.
// 3. Додає новий шлях до файлу у список FilePaths.
// 4. Перезаписує JSON-файл з оновленим списком шляхів.
//
// Повертає:
// - Помилку, якщо виникла проблема при відкритті, декодуванні або запису файлу.
func (f *AddPath) AddFilePath(filePath string, jsonFilePath string) error {
	// Читання існуючого JSON-файлу
	file, err := os.OpenFile(jsonFilePath, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	// Декодування JSON
	decoder := json.NewDecoder(file)
	var store AddPath
	if err := decoder.Decode(&store); err != nil && err != io.EOF {
		return err
	}

	// Додавання нового посилання
	store.FilePaths = append(store.FilePaths, filePath)

	// Перезапис JSON-файлу
	file.Seek(0, 0) // Переміщення курсора на початок файлу
	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	return encoder.Encode(&store)
}

//Що робить кожна частина коду:
//AddPath структура: Вона має одне поле FilePaths, яке є списком шляхів до файлів. Це поле буде зберігатися у JSON-файлі.
//
//NewAddPath функція: Створює новий екземпляр AddPath. Це конструктор для структури.
//
//AddFilePath метод:
//
//Читання JSON-файлу: Файл відкривається для читання та запису. Якщо файл не існує, він створюється.
//Декодування JSON: JSON дані зчитуються та декодуються в структуру AddPath.
//Додавання нового шляху: Новий шлях до файлу додається до списку FilePaths.
//Перезапис JSON-файлу: Файл перезаписується з новим списком шляхів. Курсор файлу переміщується на початок перед записом нового JSON.
//Повернення помилок: Метод повертає помилку, якщо виникає проблема на будь-якому з етапів (відкриття файлу, декодування, запис).
