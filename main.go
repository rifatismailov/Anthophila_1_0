package main

import (
	"Anthophila/config"
	"Anthophila/information"
	"Anthophila/logging"

	//"Anthophila/management"
	"Anthophila/checkfile"
	"fmt"
	"strings"

	"github.com/elastic/go-elasticsearch/v8"
)

func main() {

	information := information.NewInfo()

	cfg, err := config.ParseOrLoadConfig()
	if err != nil {
		fmt.Println("Config error:", err)
		return
	}

	var username, password string

	if cfg.LogCredentials != nil {
		parts := strings.SplitN(*cfg.LogCredentials, ":", 2)
		if len(parts) == 2 {
			username = parts[0]
			password = parts[1]
		} else {
			fmt.Println("⚠️ Invalid log_credentials format, expected user:pass")
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

	logger := logging.NewLoggerService(
		information.GetMACAddress(),
		information.HostName(), "elasticsearch", esClient)
	logger.LogInfo("Start Anthophila", "Start of work")

	file_checker := checkfile.NewFileChecker(*&cfg.FileServer, logger, *&cfg.Key, *&cfg.Directories, *&cfg.Extensions, int8(*&cfg.Hour), int8(*&cfg.Minute), information)
	file_checker.Start()
	// Ініціалізація та запуск Manager
	//manager := management.NewManager(logger, "ws://"+*cfg.ManagerServer+"/ws", cfg.Key)
	//manager.Start()
	select {}
}
