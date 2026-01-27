package config

import "github.com/creasty/defaults"

const (
	// DefaultGainMode is the default SDR gain mode.
	DefaultGainMode = "auto"
)

// DefaultConfig returns a configuration with default values.
func DefaultConfig() *Config {
	config := &Config{
		Meters: make([]MeterConfig, 0),
	}

	// Set default values using the defaults library
	if err := defaults.Set(config); err != nil {
		// If defaults fail, fall back to manual defaults
		config.General.SleepFor = 0
		config.General.Verbosity = "info"
		config.General.HealthCheckEnabled = true
		config.SDR.USBDevice = ""
		config.SDR.FreqCorrection = 0
		config.SDR.GainMode = DefaultGainMode
		config.SDR.Gain = 0
		config.SDR.AGCEnabled = true
		config.MQTT.Port = 1883
		config.MQTT.BaseTopic = "meters"
		config.MQTT.HomeAssistant.Enabled = true
		config.MQTT.HomeAssistant.DiscoveryPrefix = "homeassistant"
		config.MQTT.HomeAssistant.StatusTopic = "homeassistant/status"
	}

	return config
}

// ValidProtocols returns the list of supported meter protocols.
func ValidProtocols() []string {
	return []string{"scm", "scm+", "idm", "netidm", "r900", "r900bcd"}
}

// ValidDeviceClasses returns the list of valid device classes.
func ValidDeviceClasses() []string {
	return []string{"none", "current", "energy", "gas", "power", "water"}
}

// ValidStateClasses returns the list of valid state classes.
func ValidStateClasses() []string {
	return []string{"measurement", "total", "total_increasing"}
}

// ValidVerbosityLevels returns the list of valid verbosity levels.
func ValidVerbosityLevels() []string {
	return []string{"none", "error", "warning", "info", "debug"}
}

// NewMeterConfig returns a new MeterConfig with default values applied.
func NewMeterConfig() *MeterConfig {
	meter := &MeterConfig{}
	if err := defaults.Set(meter); err != nil {
		// Fallback to manual defaults
		meter.Protocol = "scm+"
		meter.Name = "Smart Meter"
		meter.UnitOfMeasurement = "kWh"
		meter.Icon = "mdi:flash"
		meter.DeviceClass = "energy"
		meter.StateClass = "total_increasing"
		meter.ExpireAfter = 0
		meter.ForceUpdate = false
	}
	return meter
}
