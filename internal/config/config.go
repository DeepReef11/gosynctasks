package config

import (
	"encoding/json"
	"fmt"
	"github.com/DeepReef11/gosynctasks/internal/utils"
	"github.com/DeepReef11/gosynctasks/connectors"
	"github.com/go-playground/validator/v10"
	"log"
	"os"
	"path/filepath"
	"sync"
)

import _ "embed"

var globalConfig *Config
var configOnce sync.Once

func GetConfig() *Config {
	configOnce.Do(func() {
		config, err := loadUserOrSampleConfig()
		if err != nil {
			log.Fatal(err)
		}
		globalConfig = config
	})
	return globalConfig
}

//go:embed config.sample.json
var sampleConfig []byte

const (
	CONFIG_DIR_PATH  = "gosynctasks"
	CONFIG_FILE_PATH = "config.json"
	CONFIG_DIR_PERM  = 0755
	CONFIG_FILE_PERM = 0644
)

type Config struct {
	Connector      connectors.TasksConnector `json:"connector"`
	CanWriteConfig bool           `json:"canWriteConfig"`
}

func (c Config) Validate() error {
	validate := validator.New()
	return validate.Struct(c)
}

func loadUserOrSampleConfig() (*Config, error) {
	return loadConfig()
}

func GetConfigPath() (string, error) {

	dir, err := os.UserConfigDir()

	if err != nil {
		return "", fmt.Errorf("failed to get user config dir: %w", err)
	}
	return filepath.Join(dir, CONFIG_DIR_PATH, CONFIG_FILE_PATH), nil
}

func createConfigDir(configPath string) error {
	return os.MkdirAll(filepath.Dir(configPath), CONFIG_DIR_PERM)
}
func WriteConfigFile(configPath string, data []byte) error {
	return os.WriteFile(configPath, data, CONFIG_FILE_PERM)
}

func createConfigFromSample(configPath string) []byte {
	var (
		configData []byte
		err        error
	)
	err = createConfigDir(configPath)
	if err != nil {
		log.Fatal(err)
	}
	configData = sampleConfig

	err = WriteConfigFile(configPath, configData)
	if err != nil {
		log.Fatal(err)
	}
	return configData
}

func loadConfig() (*Config, error) {
	var (
		configPath        string
		configData        []byte
		err               error
		canWriteConfig    bool
		noConfigFileFound = false
	)
	configPath, err = GetConfigPath()
	if err != nil {
		log.Fatalf("Config path couldn't be retrieved")
	}

	configData, err = os.ReadFile(configPath)
	if os.IsNotExist(err) {
		noConfigFileFound = true
		fmt.Println("No config exist at ", configPath)

		shouldCopySample := utils.PromptYesNo("Do you want to copy config sample to " + configPath + "?")
		if shouldCopySample {
			configData = createConfigFromSample(configPath)
			canWriteConfig = true

		} else {
			configData = sampleConfig
			canWriteConfig = false
		}
	}

	var configObj Config
	err = json.Unmarshal(configData, &configObj)

	if err != nil {
		log.Fatalf("Invalid JSON in config file %s: %v", configPath, err)
	}
	if err = configObj.Validate(); err != nil {
		log.Fatalf("Missing field(s) in JSON config file %s: %v", configPath, err)
	}
	if noConfigFileFound {
		configObj.CanWriteConfig = canWriteConfig
	}

	return &configObj, nil

}
