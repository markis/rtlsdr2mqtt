package config

import (
	"errors"
	"slices"
	"testing"

	"github.com/creasty/defaults"
)

const (
	testVerbosity         = "info"
	testBaseTopic         = "meters"
	testStateClass        = "total_increasing"
	testUnitMeasurement   = "kWh"
	testDeviceClass       = "energy"
	testHAStatusTopic     = "homeassistant/status"
	testHADiscoveryPrefix = "homeassistant"
)

func TestDefaultValues(t *testing.T) {
	config := &Config{}
	err := defaults.Set(config)
	if err != nil {
		t.Fatalf("Failed to set defaults: %v", err)
	}

	// Test General config defaults
	if config.General.Verbosity != testVerbosity {
		t.Errorf("Expected verbosity '%s', got '%s'", testVerbosity, config.General.Verbosity)
	}

	// Test SDR config defaults
	if config.SDR.GainMode != "auto" {
		t.Errorf("Expected GainMode 'auto', got '%s'", config.SDR.GainMode)
	}

	if config.SDR.AGCEnabled != true {
		t.Errorf("Expected AGCEnabled true, got %v", config.SDR.AGCEnabled)
	}

	// Test MQTT config defaults
	if config.MQTT.Port != 1883 {
		t.Errorf("Expected MQTT port 1883, got %d", config.MQTT.Port)
	}

	if config.MQTT.BaseTopic != testBaseTopic {
		t.Errorf("Expected base topic '%s', got '%s'", testBaseTopic, config.MQTT.BaseTopic)
	}
}

func TestMeterConfigDefaults(t *testing.T) {
	meter := NewMeterConfig()

	if meter.Protocol != "scm+" {
		t.Errorf("Expected protocol 'scm+', got '%s'", meter.Protocol)
	}

	if meter.StateClass != testStateClass {
		t.Errorf("Expected state class '%s', got '%s'", testStateClass, meter.StateClass)
	}

	if meter.UnitOfMeasurement != testUnitMeasurement {
		t.Errorf("Expected unit '%s', got '%s'", testUnitMeasurement, meter.UnitOfMeasurement)
	}

	if meter.DeviceClass != testDeviceClass {
		t.Errorf("Expected device class '%s', got '%s'", testDeviceClass, meter.DeviceClass)
	}

	if meter.Name != "Smart Meter" {
		t.Errorf("Expected name 'Smart Meter', got '%s'", meter.Name)
	}

	if meter.Icon != "mdi:flash" {
		t.Errorf("Expected icon 'mdi:flash', got '%s'", meter.Icon)
	}

	if meter.ExpireAfter != 0 {
		t.Errorf("Expected ExpireAfter 0, got %d", meter.ExpireAfter)
	}

	if meter.ForceUpdate {
		t.Error("Expected ForceUpdate false")
	}
}

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	// Ensure meters array is initialized
	if config.Meters == nil {
		t.Error("Expected meters array to be initialized")
	}

	// Test that defaults are applied
	if config.General.Verbosity != testVerbosity {
		t.Errorf("Expected verbosity '%s', got '%s'", testVerbosity, config.General.Verbosity)
	}

	if config.MQTT.Port != 1883 {
		t.Errorf("Expected MQTT port 1883, got %d", config.MQTT.Port)
	}

	if config.MQTT.BaseTopic != testBaseTopic {
		t.Errorf("Expected base topic '%s', got '%s'", testBaseTopic, config.MQTT.BaseTopic)
	}

	if config.MQTT.HomeAssistant.StatusTopic != testHAStatusTopic {
		t.Errorf("Expected HA status topic '%s', got '%s'", testHAStatusTopic, config.MQTT.HomeAssistant.StatusTopic)
	}

	if config.MQTT.HomeAssistant.DiscoveryPrefix != testHADiscoveryPrefix {
		t.Errorf("Expected HA discovery prefix '%s', got '%s'", testHADiscoveryPrefix, config.MQTT.HomeAssistant.DiscoveryPrefix)
	}

	if config.General.SleepFor != 0 {
		t.Errorf("Expected SleepFor 0, got %d", config.General.SleepFor)
	}

	if config.SDR.USBDevice != "" {
		t.Errorf("Expected USBDevice '', got '%s'", config.SDR.USBDevice)
	}

	if config.SDR.GainMode != "auto" {
		t.Errorf("Expected GainMode 'auto', got '%s'", config.SDR.GainMode)
	}

	if config.SDR.FreqCorrection != 0 {
		t.Errorf("Expected FreqCorrection 0, got %d", config.SDR.FreqCorrection)
	}
}

func TestGetIntFromString(t *testing.T) {
	tests := []struct {
		input        string
		defaultValue int
		expected     int
	}{
		{"", 10, 10},
		{"123", 10, 123},
		{"0", 10, 0},
		{"-5", 10, -5},
		{"invalid", 10, 10},
		{"12.34", 10, 10},
	}

	for _, tt := range tests {
		result := GetIntFromString(tt.input, tt.defaultValue)
		if result != tt.expected {
			t.Errorf("GetIntFromString(%q, %d) = %d, expected %d", tt.input, tt.defaultValue, result, tt.expected)
		}
	}
}

func TestGetBoolFromString(t *testing.T) {
	tests := []struct {
		input        string
		defaultValue bool
		expected     bool
	}{
		{"", true, true},
		{"", false, false},
		{"true", false, true},
		{"True", false, true},
		{"TRUE", false, true},
		{"1", false, true},
		{"yes", false, true},
		{"Yes", false, true},
		{"on", false, true},
		{"On", false, true},
		{"false", true, false},
		{"False", true, false},
		{"FALSE", true, false},
		{"0", true, false},
		{"no", true, false},
		{"No", true, false},
		{"off", true, false},
		{"Off", true, false},
		{"invalid", true, true},
		{"invalid", false, false},
	}

	for _, tt := range tests {
		result := GetBoolFromString(tt.input, tt.defaultValue)
		if result != tt.expected {
			t.Errorf("GetBoolFromString(%q, %v) = %v, expected %v", tt.input, tt.defaultValue, result, tt.expected)
		}
	}
}

func TestValidProtocols(t *testing.T) {
	protocols := ValidProtocols()

	if len(protocols) == 0 {
		t.Error("ValidProtocols returned empty slice")
	}

	expectedProtocols := []string{"scm", "scm+", "idm", "netidm", "r900", "r900bcd"}
	if len(protocols) != len(expectedProtocols) {
		t.Errorf("Expected %d protocols, got %d", len(expectedProtocols), len(protocols))
	}

	for _, expected := range expectedProtocols {
		if !slices.Contains(protocols, expected) {
			t.Errorf("Expected protocol %s not found in ValidProtocols", expected)
		}
	}
}

func TestValidDeviceClasses(t *testing.T) {
	classes := ValidDeviceClasses()

	if len(classes) == 0 {
		t.Error("ValidDeviceClasses returned empty slice")
	}

	expectedClasses := []string{"none", "current", "energy", "gas", "power", "water"}
	if len(classes) != len(expectedClasses) {
		t.Errorf("Expected %d device classes, got %d", len(expectedClasses), len(classes))
	}
}

func TestValidStateClasses(t *testing.T) {
	classes := ValidStateClasses()

	if len(classes) == 0 {
		t.Error("ValidStateClasses returned empty slice")
	}

	expectedClasses := []string{"measurement", "total", "total_increasing"}
	if len(classes) != len(expectedClasses) {
		t.Errorf("Expected %d state classes, got %d", len(expectedClasses), len(classes))
	}
}

func TestValidVerbosityLevels(t *testing.T) {
	levels := ValidVerbosityLevels()

	if len(levels) == 0 {
		t.Error("ValidVerbosityLevels returned empty slice")
	}

	expectedLevels := []string{"none", "error", "warning", "info", "debug"}
	if len(levels) != len(expectedLevels) {
		t.Errorf("Expected %d verbosity levels, got %d", len(expectedLevels), len(levels))
	}
}

func TestApplyDefaults(t *testing.T) {
	config := &Config{
		Meters: []MeterConfig{
			{
				ID:          "12345",
				Name:        "Test Meter",
				DeviceClass: "energy",
			},
			{
				ID:          "67890",
				Name:        "Gas Meter",
				DeviceClass: "gas",
			},
			{
				ID:          "11111",
				Name:        "Water Meter",
				DeviceClass: "water",
			},
			{
				ID:          "22222",
				Name:        "Power Meter",
				DeviceClass: "power",
			},
			{
				ID:          "33333",
				Name:        "Unknown Meter",
				DeviceClass: "unknown",
			},
		},
	}

	result := applyDefaults(config)

	// Check energy meter defaults
	if result.Meters[0].StateClass != defaultStateClass {
		t.Errorf("Expected state class %s for energy meter, got %s", defaultStateClass, result.Meters[0].StateClass)
	}
	if result.Meters[0].UnitOfMeasurement != testUnitMeasurement {
		t.Errorf("Expected unit %s for energy meter, got %s", testUnitMeasurement, result.Meters[0].UnitOfMeasurement)
	}

	// Check gas meter defaults
	if result.Meters[1].UnitOfMeasurement != "m³" {
		t.Errorf("Expected unit m³ for gas meter, got %s", result.Meters[1].UnitOfMeasurement)
	}

	// Check water meter defaults
	if result.Meters[2].UnitOfMeasurement != "L" {
		t.Errorf("Expected unit L for water meter, got %s", result.Meters[2].UnitOfMeasurement)
	}

	// Check power meter defaults
	if result.Meters[3].UnitOfMeasurement != "W" {
		t.Errorf("Expected unit W for power meter, got %s", result.Meters[3].UnitOfMeasurement)
	}

	// Check unknown device class defaults to kWh
	if result.Meters[4].UnitOfMeasurement != testUnitMeasurement {
		t.Errorf("Expected unit %s for unknown device class, got %s", testUnitMeasurement, result.Meters[4].UnitOfMeasurement)
	}
}

func TestNormalizeMeters(t *testing.T) {
	config := &Config{
		Meters: []MeterConfig{
			{
				ID:   "  12345  ",
				Name: "Test Meter",
			},
			{
				ID:         "67890",
				Name:       "Test Meter 2",
				StateClass: "measurement",
			},
		},
	}

	result := normalizeMeters(config)

	// Check ID is trimmed
	if result.Meters[0].ID != "12345" {
		t.Errorf("Expected trimmed ID '12345', got '%s'", result.Meters[0].ID)
	}

	// Check state class is set
	if result.Meters[0].StateClass != testStateClass {
		t.Errorf("Expected state class '%s', got '%s'", testStateClass, result.Meters[0].StateClass)
	}

	// Check existing state class is preserved
	if result.Meters[1].StateClass != "measurement" {
		t.Errorf("Expected state class 'measurement', got '%s'", result.Meters[1].StateClass)
	}
}

func TestValidateMeter(t *testing.T) {
	tests := []struct {
		name      string
		meter     *MeterConfig
		expectErr bool
		errType   error
	}{
		{
			name: "valid meter",
			meter: &MeterConfig{
				ID:          "12345",
				Name:        "Test Meter",
				Protocol:    "scm+",
				DeviceClass: "energy",
				StateClass:  "total_increasing",
			},
			expectErr: false,
		},
		{
			name: "empty ID",
			meter: &MeterConfig{
				ID:       "",
				Name:     "Test Meter",
				Protocol: "scm+",
			},
			expectErr: true,
			errType:   ErrMeterIDEmpty,
		},
		{
			name: "empty name",
			meter: &MeterConfig{
				ID:       "12345",
				Name:     "",
				Protocol: "scm+",
			},
			expectErr: true,
			errType:   ErrMeterNameEmpty,
		},
		{
			name: "empty protocol",
			meter: &MeterConfig{
				ID:       "12345",
				Name:     "Test Meter",
				Protocol: "",
			},
			expectErr: true,
			errType:   ErrMeterProtocolEmpty,
		},
		{
			name: "invalid protocol",
			meter: &MeterConfig{
				ID:       "12345",
				Name:     "Test Meter",
				Protocol: "invalid",
			},
			expectErr: true,
		},
		{
			name: "invalid device class",
			meter: &MeterConfig{
				ID:          "12345",
				Name:        "Test Meter",
				Protocol:    "scm+",
				DeviceClass: "invalid",
			},
			expectErr: true,
		},
		{
			name: "invalid state class",
			meter: &MeterConfig{
				ID:         "12345",
				Name:       "Test Meter",
				Protocol:   "scm+",
				StateClass: "invalid",
			},
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateMeter(tt.meter.ID, tt.meter)
			if tt.expectErr && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.expectErr && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}
			if tt.errType != nil && !errors.Is(err, tt.errType) {
				t.Logf("Expected specific error type, got: %v", err)
			}
		})
	}
}

func TestFindMeterByID(t *testing.T) {
	config := &Config{
		Meters: []MeterConfig{
			{ID: "12345", Name: "Meter 1"},
			{ID: "67890", Name: "Meter 2"},
			{ID: "11111", Name: "Meter 3"},
		},
	}

	tests := []struct {
		id        string
		found     bool
		meterName string
	}{
		{"12345", true, "Meter 1"},
		{"67890", true, "Meter 2"},
		{"11111", true, "Meter 3"},
		{"99999", false, ""},
		{"", false, ""},
	}

	for _, tt := range tests {
		t.Run(tt.id, func(t *testing.T) {
			meter, found := config.FindMeterByID(tt.id)

			if found != tt.found {
				t.Errorf("FindMeterByID(%s) found=%v, expected %v", tt.id, found, tt.found)
			}

			if found && meter.Name != tt.meterName {
				t.Errorf("FindMeterByID(%s) returned meter with name %s, expected %s", tt.id, meter.Name, tt.meterName)
			}

			if !found && meter != nil {
				t.Errorf("FindMeterByID(%s) returned non-nil meter when not found", tt.id)
			}
		})
	}
}
