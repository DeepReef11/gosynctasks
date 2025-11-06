package config

import (
	"encoding/json"
	"fmt"

	// "gosynctasks/backend"
	"gosynctasks/backend"
	// "gosynctasks/connectors"
	"gosynctasks/internal/utils"
	"log"
	"os"
	"path/filepath"
	"sync"

	"github.com/go-playground/validator/v10"

	_ "embed"
)

var configOnce sync.Once

var globalConfig *Config

// var globalConnector *connectors.TaskConnector

//go:embed config.sample.json
var sampleConfig []byte

const (
	CONFIG_DIR_PATH  = "gosynctasks"
	CONFIG_FILE_PATH = "config.json"
	CONFIG_DIR_PERM  = 0755
	CONFIG_FILE_PERM = 0644
)

type Config struct {
	Connector      backend.ConnectorConfig `json:"connector"`
	CanWriteConfig bool                    `json:"canWriteConfig"`
	UI             string                  `json:"ui" validate:"oneof=cli tui"`
	DateFormat     string                  `json:"date_format,omitempty"` // Go time format string, defaults to "2006-01-02"
}

func (c Config) Validate() error {
	validate := validator.New()
	return validate.Struct(c)
}

func (c *Config) GetDateFormat() string {
	if c.DateFormat == "" {
		return "2006-01-02" // Default to yyyy-mm-dd
	}
	return c.DateFormat
}

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

func loadUserOrSampleConfig() (*Config, error) {

	configPath, err := GetConfigPath()
	if err != nil {
		log.Fatalf("Config path couldn't be retrieved")
		return nil, err
	}
	configData, err := configDataFromPath(configPath)
	if err != nil {
		log.Fatalf("Config data couldn't be retrieved")
		return nil, err
	}
	configObj, err := parseConfig(configData, configPath)
	return configObj, err
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

func parseConfig(configData []byte, configPath string) (*Config, error) {
	var configObj Config
	// configObj, err := UnmarshalJSON(configData)
	err := json.Unmarshal(configData, &configObj)

	if err != nil {
		log.Fatalf("Invalid JSON in config file %s: %v", configPath, err)
	}
	if err = configObj.Validate(); err != nil {
		log.Fatalf("Missing field(s) in JSON config file %s: %v", configPath, err)
	}
	return &configObj, err
}

func configDataFromPath(configPath string) ([]byte, error) {
	var (
		configData []byte
		err        error
	)

	configData, err = os.ReadFile(configPath)
	if os.IsNotExist(err) {
		fmt.Println("No config exist at ", configPath)

		shouldCopySample := utils.PromptYesNo("Do you want to copy config sample to " + configPath + "?")
		if shouldCopySample {
			configData = createConfigFromSample(configPath)

		} else {
			configData = sampleConfig
		}
	}

	return configData, nil

}
