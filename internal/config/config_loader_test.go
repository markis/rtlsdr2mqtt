package config

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

const (
	testConfigVerbosity = "info"
	testConfigBaseTopic = "meters"
)

func TestLoadConfigNonExistentFile(t *testing.T) {
	_, err := LoadConfig("/nonexistent/path/to/config.yaml")

	if err == nil {
		t.Error("Expected error for nonexistent file")
	}

	if !os.IsNotExist(err) && !errors.Is(err, ErrConfigNotFound) {
		t.Logf("Got error: %v", err)
	}
}

func TestLoadConfigValidYAML(t *testing.T) {
	// Create temporary config file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "test_config.yaml")

	configContent := `
general:
  verbosity: info
  rtltcp_host: "127.0.0.1:1234"
mqtt:
  host: "localhost"
  port: 1883
  base_topic: "meters"
meters:
  - id: "12345678"
    name: "Test Meter"
    protocol: "scm+"
    device_class: "energy"
    state_class: "total_increasing"
    unit_of_measurement: "kWh"
`
	if err := os.WriteFile(configPath, []byte(configContent), 0o600); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	config, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	if config.General.Verbosity != "info" {
		t.Errorf("Expected verbosity 'info', got '%s'", config.General.Verbosity)
	}

	if config.MQTT.Host != "localhost" {
		t.Errorf("Expected MQTT host 'localhost', got '%s'", config.MQTT.Host)
	}

	if len(config.Meters) != 1 {
		t.Errorf("Expected 1 meter, got %d", len(config.Meters))
	}

	if config.Meters[0].ID != "12345678" {
		t.Errorf("Expected meter ID '12345678', got '%s'", config.Meters[0].ID)
	}
}

func TestLoadConfigValidJSON(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "test_config.json")

	configContent := `{
  "general": {
    "verbosity": "debug",
    "rtltcp_host": "192.168.1.1:1234"
  },
  "mqtt": {
    "host": "mqtt.example.com",
    "port": 8883,
    "base_topic": "meters"
  },
  "meters": [
    {
      "id": "87654321",
      "name": "JSON Meter",
      "protocol": "idm",
      "device_class": "gas",
      "state_class": "total",
      "unit_of_measurement": "m³"
    }
  ]
}`
	if err := os.WriteFile(configPath, []byte(configContent), 0o600); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	config, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	if config.General.Verbosity != "debug" {
		t.Errorf("Expected verbosity 'debug', got '%s'", config.General.Verbosity)
	}

	if config.MQTT.Host != "mqtt.example.com" {
		t.Errorf("Expected MQTT host 'mqtt.example.com', got '%s'", config.MQTT.Host)
	}

	if config.MQTT.Port != 8883 {
		t.Errorf("Expected MQTT port 8883, got %d", config.MQTT.Port)
	}

	if len(config.Meters) != 1 {
		t.Errorf("Expected 1 meter, got %d", len(config.Meters))
	}

	if config.Meters[0].Protocol != "idm" {
		t.Errorf("Expected protocol 'idm', got '%s'", config.Meters[0].Protocol)
	}
}

func TestLoadConfigInvalidYAML(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "invalid.yaml")

	invalidContent := `
general:
  verbosity: info
  invalid yaml syntax here: [[[
mqtt:
  host: "localhost"
`
	if err := os.WriteFile(configPath, []byte(invalidContent), 0o600); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	_, err := LoadConfig(configPath)
	if err == nil {
		t.Error("Expected error for invalid YAML")
	}
}

func TestLoadConfigInvalidJSON(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "invalid.json")

	invalidContent := `{
  "general": {
    "verbosity": "info",
  }
  "mqtt": {
    "host": "localhost"
  }
}`
	if err := os.WriteFile(configPath, []byte(invalidContent), 0o600); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	_, err := LoadConfig(configPath)
	if err == nil {
		t.Error("Expected error for invalid JSON")
	}
}

func TestLoadConfigUnsupportedFormat(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.txt")

	content := "some text content"
	if err := os.WriteFile(configPath, []byte(content), 0o600); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	_, err := LoadConfig(configPath)
	if err == nil {
		t.Error("Expected error for unsupported format")
	}
}

func TestLoadConfigValidationNoMeters(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "no_meters.yaml")

	configContent := `
general:
  verbosity: info
mqtt:
  host: "localhost"
  port: 1883
meters: []
`
	if err := os.WriteFile(configPath, []byte(configContent), 0o600); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	_, err := LoadConfig(configPath)
	if err == nil {
		t.Error("Expected error for config with no meters")
	}
}

func TestLoadConfigValidationNoMQTTHost(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "no_mqtt.yaml")

	configContent := `
general:
  verbosity: info
mqtt:
  port: 1883
meters:
  - id: "12345678"
    name: "Test Meter"
    protocol: "scm+"
`
	if err := os.WriteFile(configPath, []byte(configContent), 0o600); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	_, err := LoadConfig(configPath)
	if err == nil {
		t.Error("Expected error for config without MQTT host")
	}
}

func TestLoadConfigValidationInvalidVerbosity(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "invalid_verbosity.yaml")

	configContent := `
general:
  verbosity: invalid_level
mqtt:
  host: "localhost"
meters:
  - id: "12345678"
    name: "Test Meter"
    protocol: "scm+"
`
	if err := os.WriteFile(configPath, []byte(configContent), 0o600); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	_, err := LoadConfig(configPath)
	if err == nil {
		t.Error("Expected error for invalid verbosity level")
	}
}

func TestLoadConfigValidationInvalidProtocol(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "invalid_protocol.yaml")

	configContent := `
general:
  verbosity: info
mqtt:
  host: "localhost"
meters:
  - id: "12345678"
    name: "Test Meter"
    protocol: "invalid_protocol"
`
	if err := os.WriteFile(configPath, []byte(configContent), 0o600); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	_, err := LoadConfig(configPath)
	if err == nil {
		t.Error("Expected error for invalid protocol")
	}
}

func TestLoadConfigValidationInvalidDeviceClass(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "invalid_device_class.yaml")

	configContent := `
general:
  verbosity: info
mqtt:
  host: "localhost"
meters:
  - id: "12345678"
    name: "Test Meter"
    protocol: "scm+"
    device_class: "invalid_class"
`
	if err := os.WriteFile(configPath, []byte(configContent), 0o600); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	_, err := LoadConfig(configPath)
	if err == nil {
		t.Error("Expected error for invalid device class")
	}
}

func TestLoadConfigValidationInvalidStateClass(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "invalid_state_class.yaml")

	configContent := `
general:
  verbosity: info
mqtt:
  host: "localhost"
meters:
  - id: "12345678"
    name: "Test Meter"
    protocol: "scm+"
    state_class: "invalid_class"
`
	if err := os.WriteFile(configPath, []byte(configContent), 0o600); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	_, err := LoadConfig(configPath)
	if err == nil {
		t.Error("Expected error for invalid state class")
	}
}

func TestLoadConfigDefaultsApplied(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "minimal.yaml")

	// Minimal config - defaults should be applied
	configContent := `
mqtt:
  host: "localhost"
meters:
  - id: "12345678"
    name: "Test Meter"
    protocol: "scm+"
`
	if err := os.WriteFile(configPath, []byte(configContent), 0o600); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	config, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	// Check defaults are applied
	if config.General.Verbosity != testConfigVerbosity {
		t.Errorf("Expected default verbosity '%s', got '%s'", testConfigVerbosity, config.General.Verbosity)
	}

	if config.MQTT.Port != 1883 {
		t.Errorf("Expected default MQTT port 1883, got %d", config.MQTT.Port)
	}

	if config.MQTT.BaseTopic != testConfigBaseTopic {
		t.Errorf("Expected default base topic '%s', got '%s'", testConfigBaseTopic, config.MQTT.BaseTopic)
	}
}

func TestFileExists(t *testing.T) {
	tmpDir := t.TempDir()

	// Test existing file
	existingFile := filepath.Join(tmpDir, "existing.txt")
	if err := os.WriteFile(existingFile, []byte("test"), 0o600); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	if !fileExists(existingFile) {
		t.Error("fileExists returned false for existing file")
	}

	// Test non-existent file
	if fileExists(filepath.Join(tmpDir, "nonexistent.txt")) {
		t.Error("fileExists returned true for non-existent file")
	}

	// Test directory
	if fileExists(tmpDir) {
		t.Error("fileExists returned true for directory")
	}
}

func TestValidateConfigEmptyMeterID(t *testing.T) {
	config := &Config{
		MQTT: MQTTConfig{Host: "localhost"},
		Meters: []MeterConfig{
			{
				ID:       "",
				Name:     "Test Meter",
				Protocol: "scm+",
			},
		},
	}

	err := validateConfig(config)
	if err == nil {
		t.Error("Expected error for empty meter ID")
	}
}

func TestValidateConfigEmptyMeterName(t *testing.T) {
	config := &Config{
		MQTT: MQTTConfig{Host: "localhost"},
		Meters: []MeterConfig{
			{
				ID:       "12345",
				Name:     "",
				Protocol: "scm+",
			},
		},
	}

	err := validateConfig(config)
	if err == nil {
		t.Error("Expected error for empty meter name")
	}
}

func TestValidateConfigEmptyMeterProtocol(t *testing.T) {
	config := &Config{
		MQTT: MQTTConfig{Host: "localhost"},
		Meters: []MeterConfig{
			{
				ID:       "12345",
				Name:     "Test Meter",
				Protocol: "",
			},
		},
	}

	err := validateConfig(config)
	if err == nil {
		t.Error("Expected error for empty meter protocol")
	}
}
