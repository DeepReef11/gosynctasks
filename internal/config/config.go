package config

import (
	"encoding/json"
	"fmt"
	"github.com/DeepReef11/gosynctasks/internal/utils"
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
	CanWriteConfig bool `json:"canWriteConfig"` 
}

func LoadUserOrSampleConfig() {
	loadConfig()
}

func GetConfigPath() string {

	dir, dirErr := os.UserConfigDir()

	var (
		configPath string
	)
	if dirErr == nil {
		configPath = filepath.Join(dir, CONFIG_DIR_PATH, CONFIG_FILE_PATH)
	}

	return configPath
}

func createConfigFromSample(configPath string) []byte {
	var (
		configFile []byte
		err        error
	)

	if err := os.MkdirAll(filepath.Dir(configPath), 0755); err != nil {
		log.Fatal(err)
	}
	configFile = sampleConfig

	err = os.WriteFile(configPath, configFile, 0644)
	if err != nil {
		log.Fatal(err)
	}
	return configFile
}

func loadConfig() {
	var (
		configPath string
		configFile []byte
		err        error
		canWriteConfig bool
	)
	configPath = GetConfigPath()
	configFile, err = os.ReadFile(configPath)
	if os.IsNotExist(err) {
		fmt.Println("No config exist at ", configPath)

		shouldCopySample := utils.PromptYesNo("Do you want to copy config sample to " + configPath + "?")
		if shouldCopySample {
			configFile = createConfigFromSample(configPath)
			canWriteConfig = true

		} else {
			configFile = sampleConfig
			canWriteConfig = false
		}
	}

	var configObj Config
	err = json.Unmarshal(configFile, &configObj)
	if err != nil {
		log.Fatalf("Invalid JSON in config file %s: %v", configPath, err)
	}
	configObj.CanWriteConfig = canWriteConfig

	fmt.Println(configObj)
}
