package main

import (
	"Anthophila/config"
	"Anthophila/information"
	"Anthophila/logging"
	"Anthophila/management"
	"fmt"
	"strings"
	"time"

	"github.com/elastic/go-elasticsearch/v8"
)

func main() {
	cfg, err := config.ParseOrLoadConfig()
	if err != nil {
		fmt.Println("Config error:", err)
		return
	}

	fmt.Println("Parsed config:", *cfg.LogCredentials)

	var username, password string

	if cfg.LogCredentials != nil {
		parts := strings.SplitN(*cfg.LogCredentials, ":", 2)
		if len(parts) == 2 {
			username = parts[0]
			password = parts[1]
		} else {
			fmt.Println("⚠️ Неправильний формат log_credentials, очікується user:pass")
		}
	}

	esClient, err := elasticsearch.NewClient(elasticsearch.Config{
		Addresses: []string{"http://" + *cfg.LogServer},
		Username:  username,
		Password:  password,
	})
	if err != nil {
		fmt.Printf("Error loading config: %v\n", err)
	}
	macAddress := information.NewInfo().GetMACAddress()
	hostName := information.NewInfo().HostName()
	logger := logging.NewLoggerService(macAddress, hostName, "elasticsearch", esClient)
	logger.LogInfo("Запуск програми", "Початок програми")
	// Зчитування конфігурації з файлу

	// Ініціалізація та запуск Manager
	manager := management.NewManager(logger, "ws://"+*cfg.ManagerServer+"/ws", cfg.Key)
	manager.Start()

	for {
		time.Sleep(time.Second) // Затримка для основного циклу
	}

}
