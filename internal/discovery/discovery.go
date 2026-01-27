// Package discovery provides functions to generate Home Assistant auto-discovery payloads and topics for smart meters.
package discovery

import (
	"fmt"

	"rtlsdr2mqtt/internal/config"
	"rtlsdr2mqtt/internal/mqtt"
	"rtlsdr2mqtt/pkg/version"
)

// GenerateDiscoveryPayload generates a Home Assistant auto-discovery payload for a meter.
func GenerateDiscoveryPayload(baseTopic string, meter *config.MeterConfig) mqtt.HADiscoveryPayload {
	meterID := meter.ID

	payload := mqtt.HADiscoveryPayload{
		Device: mqtt.DeviceInfo{
			Identifiers:  "meter_" + meterID,
			Name:         meter.Name,
			Manufacturer: "RTLAMR2MQTT",
			Model:        "Smart Meter",
			SWVersion:    "1.0",
			SerialNumber: meterID,
			HWVersion:    "1.0",
		},
		Origin: mqtt.OriginInfo{
			Name:       "rtlsdr2mqtt",
			SWVersion:  version.Version,
			SupportURL: "https://github.com/markis/rtlsdr2mqtt",
		},
		Components:        make(map[string]mqtt.Component),
		StateTopic:        fmt.Sprintf("%s/%s/state", baseTopic, meterID),
		AvailabilityTopic: baseTopic + "/status",
		QoS:               1,
	}

	// Reading sensor component
	readingComponent := mqtt.Component{
		Platform:            "sensor",
		Name:                "Reading",
		ValueTemplate:       "{{ value_json.reading|float }}",
		JSONAttributesTopic: fmt.Sprintf("%s/%s/attributes", baseTopic, meterID),
		UniqueID:            meterID + "_reading",
	}

	// Apply meter-specific configuration
	if meter.UnitOfMeasurement != "" {
		readingComponent.UnitOfMeasurement = meter.UnitOfMeasurement
	}
	if meter.Icon != "" {
		readingComponent.Icon = meter.Icon
	}
	if meter.DeviceClass != "" && meter.DeviceClass != "none" {
		readingComponent.DeviceClass = meter.DeviceClass
	}
	if meter.StateClass != "" {
		readingComponent.StateClass = meter.StateClass
	}
	if meter.ExpireAfter > 0 {
		readingComponent.ExpireAfter = meter.ExpireAfter
	}
	if meter.ForceUpdate {
		readingComponent.ForceUpdate = true
	}

	meterReading := meterID + "_reading"
	payload.Components[meterReading] = readingComponent

	// Last seen sensor component
	meterLastSeen := meterID + "_lastseen"
	payload.Components[meterLastSeen] = mqtt.Component{
		Platform:      "sensor",
		Name:          "Last Seen",
		DeviceClass:   "timestamp",
		ValueTemplate: "{{ value_json.lastseen }}",
		UniqueID:      meterLastSeen,
	}

	return payload
}

// GenerateDiscoveryTopic generates the MQTT topic for Home Assistant discovery.
func GenerateDiscoveryTopic(haDiscoveryTopic, meterID string) string {
	return fmt.Sprintf("%s/device/%s/config", haDiscoveryTopic, meterID)
}

// GenerateStateTopic generates the MQTT topic for meter state.
func GenerateStateTopic(baseTopic, meterID string) string {
	return fmt.Sprintf("%s/%s/state", baseTopic, meterID)
}

// GenerateAttributesTopic generates the MQTT topic for meter attributes.
func GenerateAttributesTopic(baseTopic, meterID string) string {
	return fmt.Sprintf("%s/%s/attributes", baseTopic, meterID)
}

// GenerateStatusTopic generates the MQTT topic for online/offline status.
func GenerateStatusTopic(baseTopic string) string {
	return baseTopic + "/status"
}
