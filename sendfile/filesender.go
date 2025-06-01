package sendfile

import (
	"Anthophila/cryptofile"
	"Anthophila/information"
	"Anthophila/logging"
	"crypto/md5"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
)

// FILESender відповідає за відправлення файлів на сервер.
type FILESender struct {
	// Можливо, додайте тут поля для зберігання стану, якщо буде потрібно.
}

// NewFILESender створює новий екземпляр FILESender.
func NewFILESender() *FILESender {
	return &FILESender{}
}

// SenderFile відправляє файл на сервер через TCP-з'єднання.
// Встановлює з'єднання з сервером, відправляє ім'я файлу, MD5 хеш-суму файлу,
// шифрує файл та відправляє зашифрований файл на сервер.
//
// Аргументи:
// - logStatus: чи потрібно вести журнал подій
// - fileAddress: адреса сервера для відправлення файлу
// - logAddress: адреса для ведення журналу
// - filePath: шлях до файлу для відправлення
// - key: ключ для шифрування файлу
// - infoJson: додаткова інформація, яка буде додана до імені файлу
func (f *FILESender) SenderFile(logStatus bool, fileAddress, logAddress, filePath string, key []byte, infoJson string) {
	// Встановлення з'єднання з сервером
	conn, err := net.Dial("tcp", fileAddress)
	if err != nil {
		sentError := information.NewAddPath().AddFilePath(filePath, "no_sent.json")
		if sentError != nil {
			// Обробка помилки
			fmt.Println("Помилка при збереженні шляху до файлу:", sentError)
			// Можливо, потрібно повернутися або вжити інші заходи
		}
		if logStatus {
			logging.Log(logAddress, "[SenderFile] Помилка з'єднання", err.Error())
		}
		fmt.Println("Помилка з'єднання: ")
		// Якщо з'єднання з сервером не вдалося, зберігаємо шлях до файлу для повторного відправлення

		return
	}

	defer conn.Close()

	// Зміна імені файлу
	fileName := filepath.Base(filePath)
	modifiedFileName := infoJson + fileName
	fileNameBytes := []byte(modifiedFileName)

	// Переконуємось, що довжина імені файлу не перевищує 512 байт
	if len(fileNameBytes) > 512 {
		if logStatus {
			logging.Log(logAddress, "[SenderFile] Ім'я файлу занадто довге", modifiedFileName)
		}
	}

	// Доповнюємо ім'я файлу до 512 байт
	paddedFileNameBytes := make([]byte, 512)
	copy(paddedFileNameBytes, fileNameBytes)

	_, err = conn.Write(paddedFileNameBytes)
	if err != nil {
		if logStatus {
			logging.Log(logAddress, "[SenderFile] Помилка відправки імені файлу", err.Error())
		}
		return
	}

	// Відкриття файлу
	file, err := os.Open(filePath)
	if err != nil {
		if logStatus {
			logging.Log(logAddress, "[SenderFile] Помилка відкриття файлу", "{FilePath :{"+filePath+"} Err :{"+err.Error()+"}}")
		}
		return
	}
	defer file.Close()

	// Обчислення хеш-сумми файлу
	hasher := md5.New()
	_, err = io.Copy(hasher, file)
	if err != nil {
		if logStatus {
			logging.Log(logAddress, "[SenderFile] Помилка обчислення хеш-сумми для файлу", "{FilePath :{"+filePath+"} Err :{"+err.Error()+"}}")
		}
		return
	}
	fileHash := hasher.Sum(nil)

	_, err = conn.Write(fileHash)
	if err != nil {
		if logStatus {
			logging.Log(logAddress, "[SenderFile] Помилка відправки хеш-сумми файлу", err.Error())
		}
		return
	}

	// Шифрування файлу та відправлення на сервер
	file.Seek(0, 0)
	encrypt := cryptofile.NewFILEEncryptor()
	encryptedFile, err := encrypt.EncryptingFile(file, key)
	if err != nil {
		if logStatus {
			logging.Log(logAddress, "[SenderFile] Помилка шифрування файлу", "{FilePath :{"+filePath+"} Err :{"+err.Error()+"}}")
		}
		return
	}
	defer encryptedFile.Close()

	_, err = io.Copy(conn, encryptedFile)
	if err != nil {
		if logStatus {
			logging.Log(logAddress, "[SenderFile] Помилка відправки зашифрованого файлу", "{FilePath :{"+filePath+"} Err :{"+err.Error()+"}}")
		}
		return
	}

	// Видалення зашифрованого файлу локально
	err = deleteFile(encryptedFile.Name())
	if err != nil {
		if logStatus {
			logging.Log(logAddress, "[SenderFile] Помилка при видаленні зашифрованого файлу", err.Error())
		}
	}

}

// deleteFile видаляє файл за вказаним шляхом.
func deleteFile(filePath string) error {
	return os.Remove(filePath)
}

//Документація:
//FILESender:
//
//Структура для відправлення файлів на сервер.
//NewFILESender():
//
//Створює новий екземпляр FILESender.
//SenderFile:
//
//Встановлює з'єднання з сервером, відправляє ім'я файлу, MD5 хеш-суму файлу.
//Шифрує файл і відправляє зашифрований файл на сервер.
//Аргументи:
//logStatus: чи потрібно вести журнал подій.
//fileAddress: адреса сервера для відправлення файлу.
//logAddress: адреса для ведення журналу.
//filePath: шлях до файлу для відправлення.
//key: ключ для шифрування файлу.
//infoJson: додаткова інформація, яка буде додана до імені файлу.
//deleteFile(filePath string) error:
//
//Видаляє файл за вказаним шляхом.
//Пояснення:
//Перевірка довжини імені файлу: Ім'я файлу доповнюється до 512 байт. Якщо воно занадто довге, це фіксується в журналі.
//Шифрування файлу: Шифрування виконується перед відправленням.
//Видалення локального файлу: Зашифрований файл видаляється після відправлення на сервер.
