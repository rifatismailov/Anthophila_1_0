package checkfile

import (
	"Anthophila/information"
	"Anthophila/logging"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	//"time"
)

type FileExistChecker interface {
	FilePathExists(path string, stateFile string) (bool, error)
}

type FileHasher interface {
	CheckAndWriteHash(path, hashFile string) (bool, error)
}

type FileSender interface {
	Send(path string) error
}

// FileChecker - структура для планування регулярної перевірки файлів у зазначених директоріях.
// Вона надає функціональність для запуску перевірки файлів з певною періодичністю.
type FileChecker struct {
	File_server         string // Адреса сервера для відправлення файлів
	Logger              *logging.LoggerService
	Key                 string   // Ключ для шифрування файлів
	Directories         []string // Список директорій для перевірки
	SupportedExtensions []string // Список підтримуваних розширень файлів
	Hour                int8
	Minute              int8              // Час початку перевірки у форматі [година, хвилина]
	Info                *information.Info // Додаткова інформація, яка буде додана до імені файлу
	Hasher              FileHasher
}

func NewFileChecker(file_server string,
	logger *logging.LoggerService,
	key string,
	directories []string,
	se []string, h int8, m int8,
	info *information.Info) *FileChecker {
	return &FileChecker{
		File_server:         file_server,
		Logger:              logger,
		Key:                 key,
		Directories:         directories,
		SupportedExtensions: se,
		Hour:                h,
		Minute:              m,
		Info:                info,
	}

}

// Start запускає процес перевірки файлів, який буде виконуватися регулярно.
// Метод налаштовує періодичну перевірку файлів у зазначених директоріях, починаючи з часу, вказаного у TimeStart.
// Запускає перевірку у новому горутині, яка виконує перевірку кожні 5 секунд.
func (fc *FileChecker) Start() {
	fmt.Println("File_server:", fc.File_server, "Key:", fc.Key)
	fmt.Println("Directories:", fc.Directories)
	fmt.Println("SupportedExtensions:", fc.SupportedExtensions)
	fmt.Println("Hour:", fc.Hour, "Minute:", fc.Minute)
	/*
		fileChan := make(chan string, 10)                        // канал з шляхами до файлів
		encryptedChan := make(chan EncryptedFile, 10) // канал з результатами

		encryptor, err := NewFILEEncryptor([]byte(fc.Key), fileChan, encryptedChan)
		if err != nil {
			fmt.Println("Помилка ініціалізації FILEEncryptor:", err)
		}

		go encryptor.Run() // запуск у горутині
	*/

	// Вставити функцію, яка буде робити затримку до вказаного часу початку

	// Створюємо екземпляр Checker для перевірки файлів

	// Запускаємо перевірку файлів у новому горутині
	//	go func() {
	//		for {
	//			time.Sleep(5 * time.Second)

	//		}
	//	}()

	vb := &VerifyBuffer{}
	if err := vb.LoadFromFile("verified_files.json"); err != nil {
		fc.Logger.LogError("Error loading verified files", err.Error())
	}

	pendingBuffer := &PendingFilesBuffer{}
	if err := pendingBuffer.LoadFromFile("pending_files.json"); err != nil {
		fc.Logger.LogError("Error loading pending files", err.Error())
	}

	for _, dir := range fc.Directories {
		err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				fc.Logger.LogError("[Walk]", err.Error())
				return nil
			}
			if info.IsDir() || !isSupportedFileType(path, fc.SupportedExtensions) {
				return nil
			}

			changed, verify, err := vb.SaveToBuffer(path)
			if err != nil {
				fc.Logger.LogError("Error checking file", err.Error())
				return nil
			}

			if changed {
				fc.Logger.LogInfo("Файл новий або змінений", verify.Path)
				pendingBuffer.AddToBuffer(Verify{
					Path: verify.Path,
					Name: verify.Name,
					Hash: verify.Hash,
				})
			}
			return nil
		})

		if err != nil {
			fc.Logger.LogError("Directory walk error", err.Error())
		}
	}

	// Після сканування всіх директорій — спроба відправити все з буфера
	for path, file := range pendingBuffer.buffer {
		if sendFile(file) {

			pendingBuffer.RemoveFromBuffer(path)
		}
	}

	// Зберігаємо оновлений стан
	if err := pendingBuffer.SaveToFile("pending_files.json"); err != nil {
		fc.Logger.LogError("Error saving pending files", err.Error())
	}
	if err := vb.SaveToFile("verified_files.json"); err != nil {
		fc.Logger.LogError("Error saving verified files", err.Error())
	}
}

// sendFile — уявна функція, яка імітує відправку файлу
func sendFile(file Verify) bool {
	fmt.Println("Send file", file.Name)
	// Реалізація відправки файлу на сервер
	// Повертаємо true, якщо відправка успішна, або false, якщо ні
	return false // або false у реальному випадку
}

// --- Допоміжна функція ---

func isSupportedFileType(file string, supportedExtensions []string) bool {
	for _, ext := range supportedExtensions {
		if strings.HasSuffix(strings.ToLower(file), strings.ToLower(ext)) {
			return true
		}
	}
	return false
}
