package checkfile

// EncryptedFile — структура з інформацією про зашифрований файл
type EncryptedFile struct {
	OriginalPath  string // Повний шлях до оригінального файлу
	OriginalName  string
	EncryptedPath string // Шлях до зашифрованого файлу
	OriginalHash  string // MD5-хеш оригінального файлу
	EncryptedName string // Назва зашифрованого файлу
	OriginalSize  int64  // Розмір оригінального файлу
}
