package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

type Config_util struct{}

func (cu *Config_util) getUserConfigPath() string {
	dir, err := os.UserConfigDir()
	if err != nil {
		panic("cannot determine config directory")
	}
	fmt.Println("dir", dir)
	return filepath.Join(dir, "Anthophila", "config.json")
}

func (cu *Config_util) loadConfigFallback() (*Config, error) {
	path := cu.getUserConfigPath()
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, errors.New("missing required parameters and config.json")
	}
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config.json: %v", err)
	}
	return &cfg, nil
}

func (cu *Config_util) saveConfig(cfg *Config) error {
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	path := cu.getUserConfigPath()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}
