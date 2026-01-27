package mqtt

import (
	"time"
)

// StatePayload represents the MQTT state message payload.
type StatePayload struct {
	Reading  any    `json:"reading"`  // Formatted reading value
	LastSeen string `json:"lastseen"` // ISO 8601 timestamp
}

// HADiscoveryPayload represents Home Assistant auto-discovery payload.
type HADiscoveryPayload struct {
	Device            DeviceInfo           `json:"device"`
	Origin            OriginInfo           `json:"origin"`
	Components        map[string]Component `json:"components"`
	StateTopic        string               `json:"state_topic"`
	AvailabilityTopic string               `json:"availability_topic"`
	QoS               int                  `json:"qos"`
}

// DeviceInfo represents the device information in HA discovery.
type DeviceInfo struct {
	Identifiers  string `json:"identifiers"`
	Name         string `json:"name"`
	Manufacturer string `json:"manufacturer"`
	Model        string `json:"model"`
	SWVersion    string `json:"sw_version"`
	SerialNumber string `json:"serial_number"`
	HWVersion    string `json:"hw_version"`
}

// OriginInfo represents the origin information in HA discovery.
type OriginInfo struct {
	Name       string `json:"name"`
	SWVersion  string `json:"sw_version"`
	SupportURL string `json:"support_url"`
}

// Component represents a sensor component in HA discovery.
type Component struct {
	Platform            string `json:"platform"`
	Name                string `json:"name"`
	ValueTemplate       string `json:"value_template"`
	UniqueID            string `json:"unique_id"`
	JSONAttributesTopic string `json:"json_attributes_topic,omitempty"`
	DeviceClass         string `json:"device_class,omitempty"`
	StateClass          string `json:"state_class,omitempty"`
	UnitOfMeasurement   string `json:"unit_of_measurement,omitempty"`
	Icon                string `json:"icon,omitempty"`
	ExpireAfter         int    `json:"expire_after,omitempty"`
	ForceUpdate         bool   `json:"force_update,omitempty"`
}

// MessageHandler represents a callback function for handling incoming MQTT messages.
type MessageHandler func(topic string, payload []byte)

// ConnectHandler represents a callback function for handling MQTT connect events.
type ConnectHandler func()

// ConnectionLostHandler represents a callback function for handling MQTT connection lost events.
type ConnectionLostHandler func(error)

// ClientConfig represents MQTT client configuration.
type ClientConfig struct {
	Host     string
	Port     int
	Username string
	Password string
	ClientID string

	TLSEnabled  bool
	TLSInsecure bool
	TLSCA       string
	TLSCert     string
	TLSKey      string

	KeepAlive      time.Duration
	ConnectTimeout time.Duration

	// Last Will and Testament
	WillTopic   string
	WillPayload string
	WillQoS     byte
	WillRetain  bool
}

// Client interface defines the MQTT client operations.
type Client interface {
	Connect() error
	Disconnect()
	IsConnected() bool

	Publish(topic string, payload any, qos byte, retain bool) error
	Subscribe(topic string, qos byte, handler MessageHandler) error
	Unsubscribe(topic string) error

	SetOnConnectHandler(handler ConnectHandler)
	SetOnConnectionLostHandler(handler ConnectionLostHandler)
}
