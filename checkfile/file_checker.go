package checkfile

import (
	"Anthophila/information"
	"Anthophila/logging"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type FileHasher interface {
	CheckAndWriteHash(path, hashFile string) (bool, error)
}

type FileChecker struct {
	File_server         string
	Logger              *logging.LoggerService
	Key                 string
	Directories         []string
	SupportedExtensions []string
	Hour                int8
	Minute              int8
	Info                *information.Info
	Hasher              FileHasher
	wg                  sync.WaitGroup
}

func NewFileChecker(file_server string, logger *logging.LoggerService, key string, directories []string, se []string, h int8, m int8, info *information.Info) *FileChecker {
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

func (fc *FileChecker) Start() {
	fc.wg.Add(1)

	fmt.Println("File_server:", fc.File_server, "Key:", fc.Key)
	fmt.Println("Directories:", fc.Directories)
	fmt.Println("SupportedExtensions:", fc.SupportedExtensions)
	fmt.Println("Hour:", fc.Hour, "Minute:", fc.Minute)

	inputEnc := make(chan Verify)
	outputEnc := make(chan EncryptedFile)

	encryptor, err := NewFILEEncryptor([]byte(fc.Key), inputEnc, outputEnc)
	if err != nil {
		fc.Logger.LogError("Помилка ініціалізації FILEEncryptor:", err.Error())
		return
	}
	go encryptor.Run()

	vb := &VerifyBuffer{}
	if err := vb.LoadFromFile("verified_files.json"); err != nil {
		fc.Logger.LogError("Error loading verified files", err.Error())
	}

	pendingBuffer := &PendingFilesBuffer{}
	if err := pendingBuffer.LoadFromFile("pending_files.json"); err != nil {
		fc.Logger.LogError("Error loading pending files", err.Error())
	}
	serverURL := "http://192.168.88.200:8020/api/files/upload"
	fs := NewFileSender(serverURL)
	fs.Start()

	go func() {
		for result := range fs.ResultChan {
			if result.Status == "Ok" {
				fmt.Println("✅ Успішно надіслано:", "Ok:"+result.Path)
				pendingBuffer.RemoveFromBuffer(result.Path)

				// Видаляємо зашифрований файл
				_ = os.Remove(result.Path)
			} else {
				fmt.Println("❌ Помилка при надсиланні:", result.Path)
				fmt.Println("   ➤ Причина:", result.Error)
			}
		}
	}()

	// Шифрування нових або змінених файлів
	go func() {
		defer fc.wg.Done()

		for {
			fc.Logger.LogInfo("Цикл сканування запущено", "Start")

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

						// Видалити стару .enc версію, якщо існує
						_ = os.Remove(verify.Path + ".enc")

						// Надіслати файл на шифрування
						inputEnc <- verify
					}
					return nil
				})
				if err != nil {
					fc.Logger.LogError("Directory walk error", err.Error())
				}
			}

			// Збереження станів
			_ = pendingBuffer.SaveToFile("pending_files.json")
			_ = vb.SaveToFile("verified_files.json")

			time.Sleep(10 * time.Second)
		}
	}()

	// Відправка зашифрованих файлів з буфера
	go func() {
		for encryptedFile := range outputEnc {
			fc.Logger.LogInfo("Файл готовий до відправки", encryptedFile.EncryptedPath)
			fs.FileChan <- encryptedFile.EncryptedPath
			pendingBuffer.AddToBuffer(encryptedFile)
			if sendFile(encryptedFile) {
				// Видаляємо з буфера, якщо відправка успішна
				//pendingBuffer.RemoveFromBuffer(encryptedFile.EncryptedPath)

				// Видаляємо зашифрований файл
				//_ = os.Remove(encryptedFile.EncryptedPath)

				fc.Logger.LogInfo("Файл успішно відправлений", encryptedFile.EncryptedName)
			} else {
				fc.Logger.LogError("Не вдалося надіслати файл", encryptedFile.EncryptedName)
			}
		}
		_ = pendingBuffer.SaveToFile("pending_files.json")
	}()

	fc.wg.Wait()
}

func sendFile(file EncryptedFile) bool {
	fmt.Println("Send file", file.EncryptedName)

	// Тут реалізація реального відправлення файлу
	return true // ← змінити на true при реальній реалізації
}

func isSupportedFileType(file string, supportedExtensions []string) bool {
	for _, ext := range supportedExtensions {
		if strings.HasSuffix(strings.ToLower(file), strings.ToLower(ext)) {
			return true
		}
	}
	return false
}
