package utils

import (
	"encoding/json"
	"fmt"

	"gopkg.in/yaml.v3"
)

// OutputJSON marshals the provided data as indented JSON and prints it to stdout.
// Returns an error if marshaling fails.
func OutputJSON(data interface{}) error {
	jsonData, err := MarshalJSON(data)
	if err != nil {
		return err
	}
	fmt.Println(string(jsonData))
	return nil
}

// OutputYAML marshals the provided data as YAML and prints it to stdout.
// Returns an error if marshaling fails.
func OutputYAML(data interface{}) error {
	yamlData, err := MarshalYAML(data)
	if err != nil {
		return err
	}
	fmt.Print(string(yamlData))
	return nil
}

// MarshalJSON marshals the provided data as indented JSON.
// Returns the JSON bytes or an error if marshaling fails.
func MarshalJSON(data interface{}) ([]byte, error) {
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal JSON: %w", err)
	}
	return jsonData, nil
}

// MarshalYAML marshals the provided data as YAML.
// Returns the YAML bytes or an error if marshaling fails.
func MarshalYAML(data interface{}) ([]byte, error) {
	yamlData, err := yaml.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal YAML: %w", err)
	}
	return yamlData, nil
}
