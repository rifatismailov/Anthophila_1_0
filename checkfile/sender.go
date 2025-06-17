///////////////////////////////////////////////////////////////////////////////
// Package: checkfile
// Клас: FileSender
// Опис: Відповідає за відправку файлів на сервер через HTTP-запит.
//       Отримує шляхи файлів через канал FileChan, надсилає їх і повідомляє
//       результат через канал ResultChan.
///////////////////////////////////////////////////////////////////////////////

package checkfile

import (
	r "Anthophila/struct_modul"
	"bytes"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
)

// /////////////////////////////////////////////////////////////////////////////
// Структура: FileSender
//
// Поля:
// - ServerURL: адреса сервера, куди надсилаються файли.
// - Iutput_to_send_enc_file: канал, у який передаються шляхи файлів для надсилання.
// - ResultChan: канал, у який надсилається результат (успішність/помилка).
// /////////////////////////////////////////////////////////////////////////////
type FileSender struct {
	ServerURL               string        // URL сервера, куди надсилати файли
	Iutput_to_send_enc_file chan string   // Канал для отримання шляхів до файлів
	ResultChan              chan r.Result // Канал для результатів (статус, шлях, помилка)
}

// /////////////////////////////////////////////////////////////////////////////
// Функція: NewFileSender
// Створює новий екземпляр FileSender з ініціалізованими каналами.
//
// Параметри:
// - serverURL: адреса сервера, куди відправляти файли.
//
// Повертає:
// - *FileSender: вказівник на новий об'єкт FileSender.
// /////////////////////////////////////////////////////////////////////////////
func NewFileSender(serverURL string) *FileSender {
	return &FileSender{
		ServerURL:               serverURL,
		Iutput_to_send_enc_file: make(chan string),
		ResultChan:              make(chan r.Result),
	}
}

// /////////////////////////////////////////////////////////////////////////////
// Метод: Start
// Запускає горутину, яка слухає FileChan і викликає sendFile для кожного шляху.
//
// Надсилає результат (успіх чи помилка) у ResultChan.
// /////////////////////////////////////////////////////////////////////////////
func (fs *FileSender) Start() {
	go func() {
		for filePath := range fs.Iutput_to_send_enc_file {
			err := fs.sendFile(filePath)
			if err != nil {
				fs.ResultChan <- r.Result{Status: "4xx", Path: filePath, Error: err}
			} else {
				fs.ResultChan <- r.Result{Status: "201", Path: filePath}
			}
		}
	}()
}

// /////////////////////////////////////////////////////////////////////////////
// Метод: sendFile (приватний)
// Відправляє файл на сервер у форматі multipart/form-data.
//
// Параметри:
// - filePath: шлях до файлу, який потрібно надіслати.
//
// Повертає:
// - помилку, якщо вона виникла під час відправлення.
// /////////////////////////////////////////////////////////////////////////////
func (fs *FileSender) sendFile(filePath string) error {
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("не вдалося відкрити файл: %v", err)
	}
	defer file.Close()

	var requestBody bytes.Buffer
	writer := multipart.NewWriter(&requestBody)

	// Додаємо файл у multipart
	part, err := writer.CreateFormFile("file", filepath.Base(filePath))
	if err != nil {
		return fmt.Errorf("не вдалося створити multipart: %v", err)
	}

	if _, err := io.Copy(part, file); err != nil {
		return fmt.Errorf("не вдалося скопіювати вміст: %v", err)
	}

	if err := writer.Close(); err != nil {
		return fmt.Errorf("не вдалося закрити multipart writer: %v", err)
	}

	// Створюємо HTTP POST-запит
	req, err := http.NewRequest("POST", fs.ServerURL, &requestBody)
	if err != nil {
		return fmt.Errorf("не вдалося створити HTTP-запит: %v", err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	// Виконуємо запит
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("не вдалося надіслати файл: %v", err)
	}
	defer resp.Body.Close()

	// Перевірка статусу відповіді
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("сервер повернув помилку: %s", string(body))
	}

	return nil
}
