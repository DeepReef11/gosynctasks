package config

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
)

const (
	CONFIG_DIR_PATH  = "gosynctask"
	CONFIG_FILE_PATH = "config.json"
)

func getConfigPath() {
	dir, dirErr := os.UserConfigDir()
	var (
		configPath string
		config []byte
	)
	if dirErr == nil {
		configPath = filepath.Join(dir, CONFIG_DIR_PATH, CONFIG_FILE_PATH)
		var err error
		config, err = os.ReadFile(configPath)
		if os.IsNotExist(err) {
			err = os.WriteFile(configPath, data, 0644)
		} else if err != nil {
			// The user has a config file but we couldn't read it.
			// Report the error instead of ignoring their configuration.
			log.Fatal(err)
		}
	}
}
