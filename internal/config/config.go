// Package config handles loading and validating the application configuration.
package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	"github.com/creasty/defaults"
	"gopkg.in/yaml.v3"
)

// Default values constants.
const (
	defaultStateClass      = "total_increasing"
	defaultUnitMeasurement = "kWh"
)

// Static errors for better error handling.
var (
	ErrConfigNotFound     = errors.New("configuration file not found")
	ErrUnsupportedFormat  = errors.New("unsupported configuration file format")
	ErrNoMetersConfigured = errors.New("at least one meter must be configured")
	ErrMQTTHostRequired   = errors.New("MQTT host must be specified")
	ErrInvalidVerbosity   = errors.New("invalid verbosity level")
	ErrMeterIDEmpty       = errors.New("meter ID cannot be empty")
	ErrMeterNameEmpty     = errors.New("meter name cannot be empty")
	ErrMeterProtocolEmpty = errors.New("meter protocol cannot be empty")
	ErrInvalidProtocol    = errors.New("invalid meter protocol")
	ErrInvalidDeviceClass = errors.New("invalid device class")
	ErrInvalidStateClass  = errors.New("invalid state class")
	ErrFileRead           = errors.New("unable to read configuration file")
	ErrJSONFormat         = errors.New("invalid JSON format")
	ErrYAMLFormat         = errors.New("invalid YAML format")
)

// SearchPaths defines the order in which configuration files are searched.
var SearchPaths = []string{
	"/data/options.json",
	"/data/options.yaml",
	"/data/options.yml",
	"/etc/rtlsdr2mqtt.yaml",
}

// LoadConfig loads configuration from the first found config file or from the specified path.
func LoadConfig(configPath string) (*Config, error) {
	var err error

	// If no config path is provided, search for the config file in the default locations
	if configPath == "" {
		for _, path := range SearchPaths {
			if fileExists(path) {
				configPath = path
				break
			}
		}
		if configPath == "" {
			return nil, fmt.Errorf("%w in search paths: %v", ErrConfigNotFound, SearchPaths)
		}
	}

	// Check if the file exists and is readable
	if !fileExists(configPath) {
		return nil, fmt.Errorf("%w: %s", ErrConfigNotFound, configPath)
	}

	// Load the configuration based on file extension
	config, err := loadConfigFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load configuration file %s: %w", configPath, err)
	}

	// Apply defaults
	config = applyDefaults(config)

	// Get MQTT config from Supervisor if available and MQTT host is not set
	if config.MQTT.Host == "" {
		config, err = configureMQTTFromSupervisor(config)
		if err != nil {
			return nil, err
		}
	}

	// Validate the configuration
	if err := validateConfig(config); err != nil {
		return nil, fmt.Errorf("configuration validation failed: %w", err)
	}

	// Convert meters slice to map if needed and normalize meter IDs
	config = normalizeMeters(config)

	return config, nil
}

// configureMQTTFromSupervisor retrieves MQTT configuration from the Home Assistant Supervisor.
func configureMQTTFromSupervisor(config *Config) (*Config, error) {
	supervisorMQTT, err := GetMQTTFromSupervisor()
	if err != nil && !errors.Is(err, ErrSupervisorNoToken) {
		return nil, fmt.Errorf("failed to get MQTT config from supervisor: %w", err)
	}
	if supervisorMQTT != nil {
		config.MQTT = *supervisorMQTT
		// Apply default values for fields not provided by supervisor
		if config.MQTT.BaseTopic == "" {
			config.MQTT.BaseTopic = "meters"
		}
		if config.MQTT.HomeAssistant.StatusTopic == "" {
			config.MQTT.HomeAssistant.StatusTopic = "homeassistant/status"
		}
		if config.MQTT.HomeAssistant.DiscoveryPrefix == "" {
			config.MQTT.HomeAssistant.DiscoveryPrefix = "homeassistant"
		}
	}
	return config, nil
}

// loadConfigFile loads a configuration file based on its extension.
func loadConfigFile(configPath string) (*Config, error) {
	// Clean the path to prevent directory traversal
	cleanPath := filepath.Clean(configPath)

	data, err := os.ReadFile(cleanPath)
	if err != nil {
		return nil, errors.Join(ErrFileRead, err)
	}

	config := &Config{}

	// Set default values first
	if defaultsErr := defaults.Set(config); defaultsErr != nil {
		return nil, fmt.Errorf("failed to set default values: %w", defaultsErr)
	}

	ext := strings.ToLower(filepath.Ext(configPath))

	switch ext {
	case ".json", ".js":
		err = json.Unmarshal(data, config)
		if err != nil {
			return nil, errors.Join(ErrJSONFormat, err)
		}
	case ".yaml", ".yml":
		err = yaml.Unmarshal(data, config)
		if err != nil {
			return nil, errors.Join(ErrYAMLFormat, err)
		}
	default:
		return nil, fmt.Errorf("%w: %s", ErrUnsupportedFormat, ext)
	}

	return config, nil
}

// applyDefaults applies any additional default values not handled by struct tags.
func applyDefaults(config *Config) *Config {
	// Apply defaults to any newly added meters that might not have defaults set
	for i := range config.Meters {
		if config.Meters[i].StateClass == "" {
			config.Meters[i].StateClass = defaultStateClass
		}
		if config.Meters[i].UnitOfMeasurement == "" {
			// Set default based on device class
			switch config.Meters[i].DeviceClass {
			case "energy":
				config.Meters[i].UnitOfMeasurement = defaultUnitMeasurement
			case "gas":
				config.Meters[i].UnitOfMeasurement = "m³"
			case "water":
				config.Meters[i].UnitOfMeasurement = "L"
			case "power":
				config.Meters[i].UnitOfMeasurement = "W"
			default:
				config.Meters[i].UnitOfMeasurement = defaultUnitMeasurement // default fallback
			}
		}
	}

	return config
}

// validateConfig validates the configuration.
func validateConfig(config *Config) error {
	// Check required fields
	if len(config.Meters) == 0 {
		return ErrNoMetersConfigured
	}

	if config.MQTT.Host == "" {
		return ErrMQTTHostRequired
	}

	// Validate verbosity level
	if !slices.Contains(ValidVerbosityLevels(), config.General.Verbosity) {
		return fmt.Errorf("%w '%s', must be one of: %v",
			ErrInvalidVerbosity, config.General.Verbosity, ValidVerbosityLevels())
	}

	// Validate each meter
	for i := range config.Meters {
		if err := validateMeter(config.Meters[i].ID, &config.Meters[i]); err != nil {
			return fmt.Errorf("meter %s: %w", config.Meters[i].ID, err)
		}
	}

	return nil
}

// validateMeter validates a single meter configuration.
func validateMeter(_ string, meter *MeterConfig) error {
	if meter.ID == "" {
		return ErrMeterIDEmpty
	}

	if meter.Name == "" {
		return ErrMeterNameEmpty
	}

	if meter.Protocol == "" {
		return ErrMeterProtocolEmpty
	}

	if !slices.Contains(ValidProtocols(), meter.Protocol) {
		return fmt.Errorf("%w '%s', must be one of: %v",
			ErrInvalidProtocol, meter.Protocol, ValidProtocols())
	}

	if meter.DeviceClass != "" && !slices.Contains(ValidDeviceClasses(), meter.DeviceClass) {
		return fmt.Errorf("%w '%s', must be one of: %v",
			ErrInvalidDeviceClass, meter.DeviceClass, ValidDeviceClasses())
	}

	if meter.StateClass != "" && !slices.Contains(ValidStateClasses(), meter.StateClass) {
		return fmt.Errorf("%w '%s', must be one of: %v",
			ErrInvalidStateClass, meter.StateClass, ValidStateClasses())
	}

	return nil
}

// normalizeMeters ensures all meter IDs are set properly.
func normalizeMeters(config *Config) *Config {
	// Normalize each meter in the array
	for i := range config.Meters {
		// Normalize ID to string (trim whitespace)
		config.Meters[i].ID = strings.TrimSpace(config.Meters[i].ID)

		// Set default state class if not provided
		if config.Meters[i].StateClass == "" {
			config.Meters[i].StateClass = "total_increasing"
		}
	}

	return config
}

// fileExists checks if a file exists and is readable.
func fileExists(path string) bool {
	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}

// GetIntFromString converts a string to int with a default value.
func GetIntFromString(s string, defaultValue int) int {
	if s == "" {
		return defaultValue
	}

	if i, err := strconv.Atoi(s); err == nil {
		return i
	}

	return defaultValue
}

// GetBoolFromString converts a string to bool with a default value.
func GetBoolFromString(s string, defaultValue bool) bool {
	if s == "" {
		return defaultValue
	}

	switch strings.ToLower(s) {
	case "true", "1", "yes", "on":
		return true
	case "false", "0", "no", "off":
		return false
	default:
		return defaultValue
	}
}
