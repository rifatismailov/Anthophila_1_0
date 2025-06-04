// new file: configparser/configparser.go
package config

import (
	"flag"
	"os"
	"path/filepath"
	"strings"
)

const configFile = "config.json"

func ParseOrLoadConfig() (*Config, error) {
	cu := &Config_util{} // ← створення екземпляра

	fileServer := flag.String("file_server", "", "File Server address (required)")
	managerServer := flag.String("manager_server", "", "Manager Server address (optional)")
	logServer := flag.String("log_server", "", "Log Server address (optional, format host:port[:user:pass])")
	dirs := flag.String("directories", "", "Comma-separated list of directories")
	exts := flag.String("extensions", ".doc,.docx,.xls,.xlsx,.ppt,.pptx", "Comma-separated list of extensions")
	hour := flag.Int("hour", -1, "Hour (required)")
	minute := flag.Int("minute", -1, "Minute (required)")
	key := flag.String("key", "", "Encryption key (required)")

	flag.Parse()

	if *fileServer == "" || *hour < 0 || *minute < 0 || *key == "" {
		return cu.loadConfigFallback()
	}

	home, _ := os.UserHomeDir()
	var directories []string
	if *dirs == "" || *dirs == "?" {
		directories = []string{
			filepath.Join(home, "Desktop/"),
			filepath.Join(home, "Documents/"),
			filepath.Join(home, "Music/"),
			filepath.Join(home, "Public/"),
			filepath.Join(home, "Downloads/"),
		}
	} else {
		directories = strings.Split(*dirs, ",")
	}

	var logServerAddr *string
	var logCreds *string
	if *logServer != "" {
		parts := strings.Split(*logServer, ":")
		if len(parts) >= 2 {
			hostPort := strings.Join(parts[0:2], ":")
			logServerAddr = &hostPort
			if len(parts) == 4 {
				creds := parts[2] + ":" + parts[3]
				logCreds = &creds
			}
		}
	}

	cfg := &Config{
		FileServer:     *fileServer,
		ManagerServer:  nilIfEmpty(managerServer),
		LogServer:      logServerAddr,
		LogCredentials: logCreds,
		Directories:    directories,
		Extensions:     strings.Split(*exts, ","),
		Hour:           *hour,
		Minute:         *minute,
		Key:            *key,
	}

	_ = cu.saveConfig(cfg) // зберігаємо без обов'язковості
	return cfg, nil
}

func nilIfEmpty(ptr *string) *string {
	if ptr == nil || *ptr == "" {
		return nil
	}
	return ptr
}
