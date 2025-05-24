package config

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
)

import _ "embed"

//go:embed config.sample.json
var sampleConfig []byte

const (
	CONFIG_DIR_PATH  = "gosynctasks"
	CONFIG_FILE_PATH = "config.json"
)

type Config struct {
	Field1 string `json:"field1"`
	Field2 int    `json:"field2"`
}

func GetConfigPath() {
	dir, dirErr := os.UserConfigDir()
	var (
		configPath string
		configFile     []byte
	)
	if dirErr == nil {
		configPath = filepath.Join(dir, CONFIG_DIR_PATH, CONFIG_FILE_PATH)
		var err error
		configFile, err = os.ReadFile(configPath)
		if os.IsNotExist(err) {
			if err := os.MkdirAll(filepath.Dir(configPath), 0755); err != nil {
				log.Fatal(err)
			}
			configFile = sampleConfig

			err = os.WriteFile(configPath, configFile, 0644)
			if err != nil {
				log.Fatal(err)
			}
		}

		var configObj Config
		err = json.Unmarshal(configFile, &configObj)
		if err != nil {
			log.Fatalf("Invalid JSON in config file %s: %v", configPath, err)
		}
		fmt.Println(configObj)
	}
}

