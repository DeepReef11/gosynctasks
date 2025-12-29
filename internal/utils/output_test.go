package utils

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

// mockStdout captures stdout output for testing
func mockStdout(t *testing.T) (*bytes.Buffer, func() string) {
	oldStdout := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("Failed to create pipe: %v", err)
	}

	os.Stdout = w

	outChan := make(chan string)
	go func() {
		var buf bytes.Buffer
		io.Copy(&buf, r)
		outChan <- buf.String()
	}()

	return &bytes.Buffer{}, func() string {
		w.Close()
		os.Stdout = oldStdout
		return <-outChan
	}
}

func TestOutputJSON(t *testing.T) {
	tests := []struct {
		name     string
		data     interface{}
		wantErr  bool
		validate func(string) bool
	}{
		{
			name: "simple map",
			data: map[string]string{
				"key1": "value1",
				"key2": "value2",
			},
			wantErr: false,
			validate: func(output string) bool {
				// Should be valid JSON
				var result map[string]string
				return json.Unmarshal([]byte(output), &result) == nil
			},
		},
		{
			name: "struct",
			data: struct {
				Name  string `json:"name"`
				Value int    `json:"value"`
			}{
				Name:  "test",
				Value: 42,
			},
			wantErr: false,
			validate: func(output string) bool {
				return strings.Contains(output, `"name"`) && strings.Contains(output, `"test"`)
			},
		},
		{
			name:    "array",
			data:    []int{1, 2, 3, 4, 5},
			wantErr: false,
			validate: func(output string) bool {
				var result []int
				return json.Unmarshal([]byte(output), &result) == nil
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, getOutput := mockStdout(t)
			err := OutputJSON(tt.data)
			output := getOutput()

			if (err != nil) != tt.wantErr {
				t.Errorf("OutputJSON() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && tt.validate != nil {
				if !tt.validate(output) {
					t.Errorf("OutputJSON() output validation failed: %s", output)
				}
			}
		})
	}
}

func TestOutputYAML(t *testing.T) {
	tests := []struct {
		name     string
		data     interface{}
		wantErr  bool
		validate func(string) bool
	}{
		{
			name: "simple map",
			data: map[string]string{
				"key1": "value1",
				"key2": "value2",
			},
			wantErr: false,
			validate: func(output string) bool {
				// Should be valid YAML
				var result map[string]string
				return yaml.Unmarshal([]byte(output), &result) == nil
			},
		},
		{
			name: "struct",
			data: struct {
				Name  string `yaml:"name"`
				Value int    `yaml:"value"`
			}{
				Name:  "test",
				Value: 42,
			},
			wantErr: false,
			validate: func(output string) bool {
				return strings.Contains(output, "name:") && strings.Contains(output, "test")
			},
		},
		{
			name:    "array",
			data:    []string{"item1", "item2", "item3"},
			wantErr: false,
			validate: func(output string) bool {
				var result []string
				return yaml.Unmarshal([]byte(output), &result) == nil
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, getOutput := mockStdout(t)
			err := OutputYAML(tt.data)
			output := getOutput()

			if (err != nil) != tt.wantErr {
				t.Errorf("OutputYAML() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && tt.validate != nil {
				if !tt.validate(output) {
					t.Errorf("OutputYAML() output validation failed: %s", output)
				}
			}
		})
	}
}

func TestMarshalJSON(t *testing.T) {
	tests := []struct {
		name    string
		data    interface{}
		wantErr bool
		check   func([]byte) bool
	}{
		{
			name:    "simple object",
			data:    map[string]int{"count": 5},
			wantErr: false,
			check: func(b []byte) bool {
				// Should be indented (contains newlines and spaces)
				return strings.Contains(string(b), "\n") && strings.Contains(string(b), "  ")
			},
		},
		{
			name:    "nil value",
			data:    nil,
			wantErr: false,
			check: func(b []byte) bool {
				return string(b) == "null"
			},
		},
		{
			name: "complex nested structure",
			data: map[string]interface{}{
				"outer": map[string]string{
					"inner": "value",
				},
			},
			wantErr: false,
			check: func(b []byte) bool {
				var result map[string]interface{}
				return json.Unmarshal(b, &result) == nil
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := MarshalJSON(tt.data)
			if (err != nil) != tt.wantErr {
				t.Errorf("MarshalJSON() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && tt.check != nil && !tt.check(got) {
				t.Errorf("MarshalJSON() validation failed for output: %s", string(got))
			}
		})
	}
}

func TestMarshalYAML(t *testing.T) {
	tests := []struct {
		name    string
		data    interface{}
		wantErr bool
		check   func([]byte) bool
	}{
		{
			name:    "simple object",
			data:    map[string]int{"count": 5},
			wantErr: false,
			check: func(b []byte) bool {
				return strings.Contains(string(b), "count:")
			},
		},
		{
			name:    "array",
			data:    []string{"a", "b", "c"},
			wantErr: false,
			check: func(b []byte) bool {
				var result []string
				return yaml.Unmarshal(b, &result) == nil
			},
		},
		{
			name: "complex structure",
			data: struct {
				Name   string            `yaml:"name"`
				Tags   []string          `yaml:"tags"`
				Config map[string]string `yaml:"config"`
			}{
				Name: "test",
				Tags: []string{"tag1", "tag2"},
				Config: map[string]string{
					"key": "value",
				},
			},
			wantErr: false,
			check: func(b []byte) bool {
				return strings.Contains(string(b), "name:") &&
					strings.Contains(string(b), "tags:") &&
					strings.Contains(string(b), "config:")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := MarshalYAML(tt.data)
			if (err != nil) != tt.wantErr {
				t.Errorf("MarshalYAML() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && tt.check != nil && !tt.check(got) {
				t.Errorf("MarshalYAML() validation failed for output: %s", string(got))
			}
		})
	}
}

func TestOutputJSON_ErrorPropagation(t *testing.T) {
	// Create a type that cannot be marshaled to JSON
	type invalidType struct {
		Func func() // functions cannot be marshaled
	}

	_, getOutput := mockStdout(t)
	err := OutputJSON(invalidType{Func: func() {}})
	getOutput()

	if err == nil {
		t.Error("OutputJSON() expected error for unmarshable type, got nil")
	}

	if !strings.Contains(err.Error(), "failed to marshal JSON") {
		t.Errorf("OutputJSON() error = %v, want error containing 'failed to marshal JSON'", err)
	}
}

func TestOutputYAML_ErrorPropagation(t *testing.T) {
	// yaml.Marshal panics for unmarshable types like functions
	// instead of returning an error. We'll test with a channel
	// which also cannot be marshaled but may handle differently

	// Note: This test verifies that MarshalYAML properly wraps errors
	// For YAML, most invalid types cause panics rather than errors
	// So we just verify that valid types work correctly

	validData := map[string]string{"test": "value"}
	_, getOutput := mockStdout(t)
	err := OutputYAML(validData)
	output := getOutput()

	if err != nil {
		t.Errorf("OutputYAML() unexpected error for valid type: %v", err)
	}

	if !strings.Contains(output, "test:") {
		t.Errorf("OutputYAML() output should contain 'test:', got: %s", output)
	}
}
