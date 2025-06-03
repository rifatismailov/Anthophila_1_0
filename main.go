package main

import (
	"Anthophila/logging"
	"Anthophila/management"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// стандартні параметри під час запуску програми
var (
	fileServer       = flag.String("file_server", "localhost:9090", "File Server address")
	managerServer    = flag.String("manager_server", "localhost:8080", "Manager Server address")
	logServer        = flag.String("log_server", "localhost:7070", "Log Server address")
	directories      = flag.String("directories", "", "Comma-separated list of directories")
	extensions       = flag.String("extensions", ".doc,.docx,.xls,.xlsx,.ppt,.pptx", "Comma-separated list of extensions")
	hour             = flag.Int("hour", 12, "Hour")
	minute           = flag.Int("minute", 30, "Minute")
	key              = flag.String("key", "a very very very very secret key", "Encryption key")
	logFileStatus    = flag.Bool("log_file_status", false, "Log File Status")
	logManagerStatus = flag.Bool("log_manager_status", false, "Log Manager Status")
	managerEnabled   = flag.Bool("manager_enabled", false, "Enable Manager Server")
)

func main() {
	flag.Parse()
	var logger = logging.NewLoggerService("console") // або IP лог-сервера
	logger.Log("Запуск програми", "")
	// Зчитування конфігурації з файлу
	config, err := loadConfig()
	if err != nil {
		fmt.Printf("Error loading config: %v\n", err)
		return
	}

	// Отримання домашньої директорії користувача
	homeDir, err := os.UserHomeDir()
	if err != nil {
		fmt.Printf("Error getting home directory: %v\n", err)
		return
	}
	// Формування списку директорій
	var dirs []string
	// Якщо параметр directories пустий або має значення "?", додаємо стандартні користувацькі директорії
	if *directories == "" || *directories == "?" {
		dirs = []string{
			filepath.Join(homeDir, "Desktop/"),
			filepath.Join(homeDir, "Documents/"),
			filepath.Join(homeDir, "Music/"),
			filepath.Join(homeDir, "Public/"),
			filepath.Join(homeDir, "Downloads/"),
		}
	} else {
		// Інакше використовуємо директорії, передані як параметр
		dirs = strings.Split(*directories, ",")
	}
	// Створення нового об'єкта конфігурації на основі параметрів командного рядка

	newConfig := &Config{
		FileServer:       *fileServer,
		ManagerServer:    *managerServer,
		LogServer:        *logServer,
		Directories:      dirs,
		Extensions:       strings.Split(*extensions, ","),
		Hour:             *hour,
		Minute:           *minute,
		Key:              *key,
		LogFileStatus:    *logFileStatus,
		LogManagerStatus: *logManagerStatus,
		ManagerEnabled:   *managerEnabled,
	}

	fmt.Println("File Server Address:", newConfig.FileServer)
	fmt.Println("Manager Server Address:", newConfig.ManagerServer)
	fmt.Println("Log Server Address:", newConfig.LogServer)
	fmt.Println("Directories:", newConfig.Directories)
	fmt.Println("Extensions:", newConfig.Extensions)
	fmt.Println("Hour:", newConfig.Hour)
	fmt.Println("Minute:", newConfig.Minute)
	fmt.Println("Key:", newConfig.Key)

	// Порівняння існуючої конфігурації з новою конфігурацією
	if config == nil ||
		config.FileServer != newConfig.FileServer ||
		config.ManagerServer != newConfig.ManagerServer ||
		config.LogServer != newConfig.LogServer ||
		strings.Join(config.Directories, ",") != strings.Join(newConfig.Directories, ",") ||
		strings.Join(config.Extensions, ",") != strings.Join(newConfig.Extensions, ",") ||
		config.Hour != newConfig.Hour ||
		config.Minute != newConfig.Minute ||
		config.Key != newConfig.Key || config.LogFileStatus != newConfig.LogFileStatus ||
		config.LogManagerStatus != newConfig.LogManagerStatus || config.ManagerEnabled != newConfig.ManagerEnabled {

		if err := saveConfig(newConfig); err != nil {
			fmt.Printf("Error saving config: %v\n", err)
			logger.Log("Програма завершується", "")
			logger.Close()
			return
		}
	}
	// Ініціалізація та запуск Manager
	serverAddr := "ws://" + newConfig.ManagerServer + "/ws"
	manager := management.NewManager(logger, serverAddr, newConfig.Key)
	manager.Start()

	for {
		fmt.Println("Main goroutine continues...")
		time.Sleep(time.Second) // Затримка для основного циклу
	}

}
