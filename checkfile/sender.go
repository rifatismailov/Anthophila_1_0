package checkfile

import (
	"bytes"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
)

// FileSender відповідає за надсилання файлів через канал
type FileSender struct {
	ServerURL  string
	FileChan   chan string // Канал, в який передають шляхи до файлів
	ResultChan chan Result // Канал для результатів відправки
}

// NewFileSender створює новий екземпляр FileSender
func NewFileSender(serverURL string) *FileSender {
	return &FileSender{
		ServerURL:  serverURL,
		FileChan:   make(chan string),
		ResultChan: make(chan Result),
	}
}

// Start запускає горутину для обробки файлів
func (fs *FileSender) Start() {
	go func() {
		for filePath := range fs.FileChan {
			err := fs.sendFile(filePath)
			if err != nil {
				fs.ResultChan <- Result{Status: "Error", Path: filePath, Error: err}
			} else {
				fs.ResultChan <- Result{Status: "Ok", Path: filePath}
			}
		}
	}()
}

// sendFile – приватний метод для надсилання файлу
func (fs *FileSender) sendFile(filePath string) error {
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("не вдалося відкрити файл: %v", err)
	}
	defer file.Close()

	var requestBody bytes.Buffer
	writer := multipart.NewWriter(&requestBody)

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

	req, err := http.NewRequest("POST", fs.ServerURL, &requestBody)
	if err != nil {
		return fmt.Errorf("не вдалося створити HTTP-запит: %v", err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("не вдалося надіслати файл: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("сервер повернув помилку: %s", string(body))
	}

	return nil
}
