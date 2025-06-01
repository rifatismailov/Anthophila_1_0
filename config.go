package main

import (
	"encoding/json"
	"io"
	"os"
)

// Config структура для зберігання конфігураційних параметрів.
type Config struct {
	FileServer       string   `json:"file_server"`      // Адреса файлового сервера
	ManagerServer    string   `json:"manager_server"`   // Адреса сервера менеджера
	LogServer        string   `json:"log_server"`       // Адреса сервера логування
	Directories      []string `json:"directories"`      // Список директорій для перевірки
	Extensions       []string `json:"extensions"`       // Список розширень файлів для перевірки
	Hour             int      `json:"hour"`             // Година запуску
	Minute           int      `json:"minute"`           // Хвилина запуску
	Key              string   `json:"key"`              // Ключ для шифрування
	LogFileStatus    bool     `json:"logFileStatus"`    // Статус логування Роботи з Файлами
	LogManagerStatus bool     `json:"LogManagerStatus"` // Статус логування під час віддаленого керування
	ManagerEnabled   bool     `json:"manager_enabled"`  // Включає віддалений доступ до пристрою
}

const configFile = "config.json"

// loadConfig зчитує конфігураційний файл у форматі JSON. Якщо файл не існує або виникає помилка при читанні, повертає помилку.

func loadConfig() (*Config, error) {
	file, err := os.Open(configFile)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		return nil, err
	}

	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, err
	}

	return &config, nil
}

func saveConfig(config *Config) error {
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(configFile, data, 0644)
}
