package config

// Config represents the complete application configuration.
type Config struct {
	General GeneralConfig `json:"general" yaml:"general"`
	SDR     SDRConfig     `json:"sdr"     yaml:"sdr"`
	MQTT    MQTTConfig    `json:"mqtt"    yaml:"mqtt"`
	Meters  []MeterConfig `json:"meters"  yaml:"meters"`
}

// GeneralConfig holds general application settings.
type GeneralConfig struct {
	SleepFor           int    `json:"sleep_for"            yaml:"sleep_for"            default:"0"`
	Verbosity          string `json:"verbosity"            yaml:"verbosity"            default:"info"`
	HealthCheckEnabled bool   `json:"health_check_enabled" yaml:"health_check_enabled" default:"true"`
}

// SDRConfig holds SDR device configuration.
// USBDevice is a USB device ID in BUS:DEV format, or empty string for auto-detect.
type SDRConfig struct {
	USBDevice      string `json:"usb_device"      yaml:"usb_device"      default:""`
	FreqCorrection int    `json:"freq_correction" yaml:"freq_correction" default:"0"`    // PPM correction
	GainMode       string `json:"gain_mode"       yaml:"gain_mode"       default:"auto"` // auto or manual
	Gain           int    `json:"gain"            yaml:"gain"            default:"0"`    // tenths of dB (e.g., 496 = 49.6 dB)
	AGCEnabled     bool   `json:"agc_enabled"     yaml:"agc_enabled"     default:"true"` // RTL2832 AGC
}

// TLSConfig holds TLS/SSL configuration for MQTT connections.
type TLSConfig struct {
	Enabled  bool   `json:"enabled"  yaml:"enabled"  default:"false"`
	Insecure bool   `json:"insecure" yaml:"insecure" default:"false"`
	CA       string `json:"ca"       yaml:"ca"       default:""`
	Cert     string `json:"cert"     yaml:"cert"     default:""`
	Keyfile  string `json:"keyfile"  yaml:"keyfile"  default:""`
}

// HomeAssistantConfig holds Home Assistant integration settings.
type HomeAssistantConfig struct {
	Enabled         bool   `json:"enabled"          yaml:"enabled"          default:"true"`
	DiscoveryPrefix string `json:"discovery_prefix" yaml:"discovery_prefix" default:"homeassistant"`
	StatusTopic     string `json:"status_topic"     yaml:"status_topic"     default:"homeassistant/status"`
}

// MQTTConfig holds MQTT broker configuration.
type MQTTConfig struct {
	Host          string              `json:"host"          yaml:"host"          default:""`
	Port          int                 `json:"port"          yaml:"port"          default:"1883"`
	User          string              `json:"user"          yaml:"user"          default:""`
	Password      string              `json:"password"      yaml:"password"      default:""`
	TLS           TLSConfig           `json:"tls"           yaml:"tls"`
	BaseTopic     string              `json:"base_topic"    yaml:"base_topic"    default:"meters"`
	HomeAssistant HomeAssistantConfig `json:"homeassistant" yaml:"homeassistant"`
}

// MeterConfig represents a single meter configuration.
type MeterConfig struct {
	ID                string `json:"id"                            yaml:"id"                            default:""`
	Protocol          string `json:"protocol"                      yaml:"protocol"                      default:"scm+"`
	Name              string `json:"name"                          yaml:"name"                          default:"Smart Meter"`
	Format            string `json:"format,omitempty"              yaml:"format,omitempty"              default:""`
	UnitOfMeasurement string `json:"unit_of_measurement,omitempty" yaml:"unit_of_measurement,omitempty" default:"kWh"`
	Icon              string `json:"icon,omitempty"                yaml:"icon,omitempty"                default:"mdi:flash"`
	DeviceClass       string `json:"device_class,omitempty"        yaml:"device_class,omitempty"        default:"energy"`
	StateClass        string `json:"state_class,omitempty"         yaml:"state_class,omitempty"         default:"total_increasing"`
	ExpireAfter       int    `json:"expire_after,omitempty"        yaml:"expire_after,omitempty"        default:"0"`
	ForceUpdate       bool   `json:"force_update,omitempty"        yaml:"force_update,omitempty"        default:"false"`
}

// FindMeterByID finds a meter configuration by its ID.
func (c *Config) FindMeterByID(id string) (*MeterConfig, bool) {
	for i := range c.Meters {
		if c.Meters[i].ID == id {
			return &c.Meters[i], true
		}
	}
	return nil, false
}
