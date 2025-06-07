package checkfile

// Структура для результатів
type Result struct {
	Status string // Наприклад "Ok" або "Error"
	Path   string // Повний шлях до файлу
	Error  error  // Якщо є помилка
}
