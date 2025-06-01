package checkfile

import (
	"Anthophila/information"
	"Anthophila/logging"
	"Anthophila/sendfile"
	"os"
	"path/filepath"
	"strings"
)

// Checker - структура для перевірки файлів у зазначених директоріях.
type Checker struct {
	FileAddress         string   // Адреса сервера для відправлення файлів
	LogAddress          string   // Адреса для ведення журналу
	Key                 []byte   // Ключ для шифрування файлів
	Directories         []string // Список директорій для перевірки
	SupportedExtensions []string // Список підтримуваних розширень файлів
	InfoJson            string   // Додаткова інформація, яка буде додана до імені файлу
	LogStatus           bool     // Вказує, чи потрібно вести журнал подій
}

// CheckFile перевіряє файли у вказаних директоріях, обчислює їх хеш-сумми та відправляє їх на сервер,
// якщо хеш-сумма змінилася або якщо файли не були відправлені раніше.
//
// Метод проходить по всіх зазначених директоріях, перевіряє типи файлів, порівнює їх хеш-сумми з
// збереженими значеннями, і в разі змін або наявності невідправлених файлів відправляє їх на сервер.
func (c *Checker) CheckFile() {
	// Завантаження списку помилок
	errorPaths, err := logging.LoadErrorPaths()
	if err != nil {
		if c.LogStatus {
			logging.Log(c.LogAddress, "[CheckFile] Помилка завантаження списку помилок", err.Error())
		}
		return
	}

	// Проходження по всіх вказаних директоріях
	for _, dir := range c.Directories {
		if logging.IsPathInErrorList(dir, errorPaths) {
			// Пропускаємо директорії з попередньою помилкою доступу
			continue
		}

		err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				if c.LogStatus {
					logging.Log(c.LogAddress,
						"[CheckFile] Помилка доступу до шляху",
						"{Path :{"+path+"} Err :{"+err.Error()+"}}")
				}

				logging.AddErrorPath(path, err.Error(), errorPaths)
				logging.SaveErrorPaths(errorPaths) // Зберігаємо оновлений список помилок
				return nil
			}

			// Перевірка, чи підтримується тип файлу
			if !info.IsDir() && isSupportedFileType(path, c.SupportedExtensions) {
				changed, errorFileInfo := NewFileInfo().CheckAndWriteHash(path, "hashes.json")
				if errorFileInfo != nil {
					if c.LogStatus {
						logging.Log(c.LogAddress,
							"[CheckFile] Помилка під час перевірки підтримування типу файлу", path)
					}
				} else if changed {
					// Хеш файлу змінився
					sendfile.NewFILESender().SenderFile(c.LogStatus, c.FileAddress, c.LogAddress, path, c.Key, c.InfoJson)
				} else {
					// Перевіряємо, чи існує посилання на файл
					exists, err := information.NewFileExist().FilePathExists(path, "no_sent.json")
					if err != nil {
						// Логування помилки перевірки існування файлу
					}
					if exists {
						sendfile.NewFILESender().SenderFile(c.LogStatus, c.FileAddress, c.LogAddress, path, c.Key, c.InfoJson)
					}
				}
			}
			return nil
		})

		if err != nil {
			if c.LogStatus {
				logging.Log(c.LogAddress,
					"[CheckFile] Помилка обходу шляху",
					"{Dir :{"+dir+"} Err :{"+err.Error()+"}}")
			}
		}
	}
}

// isSupportedFileType перевіряє, чи має файл одне з підтримуваних розширень.
//
// Аргументи:
// - file: шлях до файлу
// - supportedExtensions: список підтримуваних розширень файлів
//
// Повертає true, якщо розширення файлу є одним з підтримуваних, інакше false.
func isSupportedFileType(file string, supportedExtensions []string) bool {
	for _, ext := range supportedExtensions {
		if strings.HasSuffix(file, ext) {
			return true
		}
	}
	return false
}
