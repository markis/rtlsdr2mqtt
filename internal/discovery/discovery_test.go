package discovery

import (
	"strings"
	"testing"

	"rtlsdr2mqtt/internal/config"
	"rtlsdr2mqtt/internal/mqtt"
	"rtlsdr2mqtt/pkg/version"
)

func TestGenerateDiscoveryPayload(t *testing.T) {
	meter := &config.MeterConfig{
		ID:                "12345678",
		Name:              "Test Meter",
		UnitOfMeasurement: "kWh",
		Icon:              "mdi:meter-electric",
		DeviceClass:       "energy",
		StateClass:        "total_increasing",
		ExpireAfter:       3600,
		ForceUpdate:       true,
	}

	baseTopic := "rtlamr"
	payload := GenerateDiscoveryPayload(baseTopic, meter)

	t.Run("device_info", func(t *testing.T) {
		assertDeviceInfo(t, &payload, meter)
	})

	t.Run("origin_info", func(t *testing.T) {
		assertOriginInfo(t, &payload)
	})

	t.Run("topics", func(t *testing.T) {
		assertTopics(t, &payload)
	})

	t.Run("components", func(t *testing.T) {
		assertComponents(t, &payload, meter)
	})
}

func assertDeviceInfo(t *testing.T, payload *mqtt.HADiscoveryPayload, meter *config.MeterConfig) {
	t.Helper()
	if payload.Device.Name != meter.Name {
		t.Errorf("Expected device name '%s', got %s", meter.Name, payload.Device.Name)
	}
	if payload.Device.Identifiers != "meter_12345678" {
		t.Errorf("Expected identifiers 'meter_12345678', got %s", payload.Device.Identifiers)
	}
	if payload.Device.SerialNumber != "12345678" {
		t.Errorf("Expected serial number '12345678', got %s", payload.Device.SerialNumber)
	}
	if payload.Device.Manufacturer != "RTLAMR2MQTT" {
		t.Errorf("Expected manufacturer 'RTLAMR2MQTT', got %s", payload.Device.Manufacturer)
	}
}

func assertOriginInfo(t *testing.T, payload *mqtt.HADiscoveryPayload) {
	t.Helper()
	if payload.Origin.Name != "rtlsdr2mqtt" {
		t.Errorf("Expected origin name 'rtlsdr2mqtt', got %s", payload.Origin.Name)
	}
	if payload.Origin.SWVersion != version.Version {
		t.Errorf("Expected origin version '%s', got %s", version.Version, payload.Origin.SWVersion)
	}
}

func assertTopics(t *testing.T, payload *mqtt.HADiscoveryPayload) {
	t.Helper()
	expectedStateTopic := "rtlamr/12345678/state"
	if payload.StateTopic != expectedStateTopic {
		t.Errorf("Expected state topic '%s', got %s", expectedStateTopic, payload.StateTopic)
	}
	expectedAvailabilityTopic := "rtlamr/status"
	if payload.AvailabilityTopic != expectedAvailabilityTopic {
		t.Errorf("Expected availability topic '%s', got %s", expectedAvailabilityTopic, payload.AvailabilityTopic)
	}
	if payload.QoS != 1 {
		t.Errorf("Expected QoS 1, got %d", payload.QoS)
	}
}

func assertComponents(t *testing.T, payload *mqtt.HADiscoveryPayload, meter *config.MeterConfig) {
	t.Helper()
	if len(payload.Components) != 2 {
		t.Errorf("Expected 2 components, got %d", len(payload.Components))
	}

	assertReadingComponent(t, payload, meter)
	assertLastSeenComponent(t, payload)
}

func assertReadingComponent(t *testing.T, payload *mqtt.HADiscoveryPayload, meter *config.MeterConfig) {
	t.Helper()
	readingKey := "12345678_reading"
	reading, ok := payload.Components[readingKey]
	if !ok {
		t.Fatalf("Expected reading component with key '%s'", readingKey)
	}

	if reading.Platform != "sensor" {
		t.Errorf("Expected platform 'sensor', got %s", reading.Platform)
	}
	if reading.Name != "Reading" {
		t.Errorf("Expected name 'Reading', got %s", reading.Name)
	}
	if reading.UnitOfMeasurement != meter.UnitOfMeasurement {
		t.Errorf("Expected unit '%s', got %s", meter.UnitOfMeasurement, reading.UnitOfMeasurement)
	}
	if reading.Icon != meter.Icon {
		t.Errorf("Expected icon '%s', got %s", meter.Icon, reading.Icon)
	}
	if reading.DeviceClass != meter.DeviceClass {
		t.Errorf("Expected device class '%s', got %s", meter.DeviceClass, reading.DeviceClass)
	}
	if reading.StateClass != meter.StateClass {
		t.Errorf("Expected state class '%s', got %s", meter.StateClass, reading.StateClass)
	}
	if reading.ExpireAfter != meter.ExpireAfter {
		t.Errorf("Expected expire after %d, got %d", meter.ExpireAfter, reading.ExpireAfter)
	}
	if !reading.ForceUpdate {
		t.Error("Expected force update to be true")
	}
	if reading.UniqueID != "12345678_reading" {
		t.Errorf("Expected unique ID '12345678_reading', got %s", reading.UniqueID)
	}
	expectedAttrTopic := "rtlamr/12345678/attributes"
	if reading.JSONAttributesTopic != expectedAttrTopic {
		t.Errorf("Expected attributes topic '%s', got %s", expectedAttrTopic, reading.JSONAttributesTopic)
	}
}

func assertLastSeenComponent(t *testing.T, payload *mqtt.HADiscoveryPayload) {
	t.Helper()
	lastSeenKey := "12345678_lastseen"
	lastSeen, ok := payload.Components[lastSeenKey]
	if !ok {
		t.Fatalf("Expected last seen component with key '%s'", lastSeenKey)
	}

	if lastSeen.Platform != "sensor" {
		t.Errorf("Expected platform 'sensor', got %s", lastSeen.Platform)
	}
	if lastSeen.Name != "Last Seen" {
		t.Errorf("Expected name 'Last Seen', got %s", lastSeen.Name)
	}
	if lastSeen.DeviceClass != "timestamp" {
		t.Errorf("Expected device class 'timestamp', got %s", lastSeen.DeviceClass)
	}
	if lastSeen.UniqueID != "12345678_lastseen" {
		t.Errorf("Expected unique ID '12345678_lastseen', got %s", lastSeen.UniqueID)
	}
}

func TestGenerateDiscoveryPayloadMinimalConfig(t *testing.T) {
	meter := &config.MeterConfig{
		ID:   "99999999",
		Name: "Minimal Meter",
	}

	baseTopic := "test"
	payload := GenerateDiscoveryPayload(baseTopic, meter)

	if payload.Device.Name != "Minimal Meter" {
		t.Errorf("Expected device name 'Minimal Meter', got %s", payload.Device.Name)
	}

	// Check that reading component exists with defaults
	readingKey := "99999999_reading"
	reading, ok := payload.Components[readingKey]
	if !ok {
		t.Fatalf("Expected reading component with key '%s'", readingKey)
	}

	// These should be empty/default since not specified
	if reading.UnitOfMeasurement != "" {
		t.Errorf("Expected empty unit, got %s", reading.UnitOfMeasurement)
	}

	if reading.Icon != "" {
		t.Errorf("Expected empty icon, got %s", reading.Icon)
	}

	if reading.ExpireAfter != 0 {
		t.Errorf("Expected expire after 0, got %d", reading.ExpireAfter)
	}

	if reading.ForceUpdate {
		t.Error("Expected force update to be false")
	}
}

func TestGenerateDiscoveryPayloadDeviceClassNone(t *testing.T) {
	meter := &config.MeterConfig{
		ID:          "11111111",
		Name:        "Test Meter",
		DeviceClass: "none",
	}

	baseTopic := "test"
	payload := GenerateDiscoveryPayload(baseTopic, meter)

	readingKey := "11111111_reading"
	reading := payload.Components[readingKey]

	// "none" device class should not be set
	if reading.DeviceClass != "" {
		t.Errorf("Expected empty device class for 'none', got %s", reading.DeviceClass)
	}
}

func TestGenerateDiscoveryTopic(t *testing.T) {
	tests := []struct {
		haDiscoveryTopic string
		meterID          string
		expected         string
	}{
		{
			haDiscoveryTopic: "homeassistant",
			meterID:          "12345678",
			expected:         "homeassistant/device/12345678/config",
		},
		{
			haDiscoveryTopic: "ha",
			meterID:          "99999999",
			expected:         "ha/device/99999999/config",
		},
		{
			haDiscoveryTopic: "custom/discovery",
			meterID:          "11223344",
			expected:         "custom/discovery/device/11223344/config",
		},
	}

	for _, tt := range tests {
		result := GenerateDiscoveryTopic(tt.haDiscoveryTopic, tt.meterID)
		if result != tt.expected {
			t.Errorf("GenerateDiscoveryTopic(%s, %s) = %s, expected %s",
				tt.haDiscoveryTopic, tt.meterID, result, tt.expected)
		}
	}
}

func TestGenerateStateTopic(t *testing.T) {
	tests := []struct {
		baseTopic string
		meterID   string
		expected  string
	}{
		{
			baseTopic: "rtlamr",
			meterID:   "12345678",
			expected:  "rtlamr/12345678/state",
		},
		{
			baseTopic: "test/topic",
			meterID:   "99999999",
			expected:  "test/topic/99999999/state",
		},
	}

	for _, tt := range tests {
		result := GenerateStateTopic(tt.baseTopic, tt.meterID)
		if result != tt.expected {
			t.Errorf("GenerateStateTopic(%s, %s) = %s, expected %s",
				tt.baseTopic, tt.meterID, result, tt.expected)
		}
	}
}

func TestGenerateAttributesTopic(t *testing.T) {
	tests := []struct {
		baseTopic string
		meterID   string
		expected  string
	}{
		{
			baseTopic: "rtlamr",
			meterID:   "12345678",
			expected:  "rtlamr/12345678/attributes",
		},
		{
			baseTopic: "test/topic",
			meterID:   "99999999",
			expected:  "test/topic/99999999/attributes",
		},
	}

	for _, tt := range tests {
		result := GenerateAttributesTopic(tt.baseTopic, tt.meterID)
		if result != tt.expected {
			t.Errorf("GenerateAttributesTopic(%s, %s) = %s, expected %s",
				tt.baseTopic, tt.meterID, result, tt.expected)
		}
	}
}

func TestGenerateStatusTopic(t *testing.T) {
	tests := []struct {
		baseTopic string
		expected  string
	}{
		{
			baseTopic: "rtlamr",
			expected:  "rtlamr/status",
		},
		{
			baseTopic: "test/topic",
			expected:  "test/topic/status",
		},
	}

	for _, tt := range tests {
		result := GenerateStatusTopic(tt.baseTopic)
		if result != tt.expected {
			t.Errorf("GenerateStatusTopic(%s) = %s, expected %s",
				tt.baseTopic, result, tt.expected)
		}
	}
}

func TestGenerateDiscoveryPayloadValueTemplate(t *testing.T) {
	meter := &config.MeterConfig{
		ID:   "12345678",
		Name: "Test Meter",
	}

	payload := GenerateDiscoveryPayload("rtlamr", meter)
	reading := payload.Components["12345678_reading"]

	expectedTemplate := "{{ value_json.reading|float }}"
	if reading.ValueTemplate != expectedTemplate {
		t.Errorf("Expected value template '%s', got '%s'", expectedTemplate, reading.ValueTemplate)
	}
}

func TestGenerateDiscoveryPayloadLastSeenValueTemplate(t *testing.T) {
	meter := &config.MeterConfig{
		ID:   "12345678",
		Name: "Test Meter",
	}

	payload := GenerateDiscoveryPayload("rtlamr", meter)
	lastSeen := payload.Components["12345678_lastseen"]

	expectedTemplate := "{{ value_json.lastseen }}"
	if lastSeen.ValueTemplate != expectedTemplate {
		t.Errorf("Expected value template '%s', got '%s'", expectedTemplate, lastSeen.ValueTemplate)
	}
}

func TestGenerateDiscoveryPayloadSupportURL(t *testing.T) {
	meter := &config.MeterConfig{
		ID:   "12345678",
		Name: "Test Meter",
	}

	payload := GenerateDiscoveryPayload("rtlamr", meter)

	expectedURL := "https://github.com/markis/rtlsdr2mqtt"
	if payload.Origin.SupportURL != expectedURL {
		t.Errorf("Expected support URL '%s', got '%s'", expectedURL, payload.Origin.SupportURL)
	}

	if !strings.HasPrefix(payload.Origin.SupportURL, "https://") {
		t.Error("Support URL should use HTTPS")
	}
}
