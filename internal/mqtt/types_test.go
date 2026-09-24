package mqtt

import (
	"encoding/json"
	"log/slog"
	"testing"
	"time"
)

const (
	testLastSeen    = "2024-01-01T12:00:00Z"
	testIdentifiers = "meter_12345"
	testMeterName   = "Test Meter"
	testSWVersion   = "1.0.0"
	testHWVersion   = "1.0"
	testOriginName  = "rtlsdr2mqtt"
	testSupportURL  = "https://github.com/test/test"
	testPlatform    = "sensor"
	testDeviceClass = "energy"
	testStateClass  = "total_increasing"
	testClientID    = "test-client"
)

func TestNewClientAppliesDefaultTimeouts(t *testing.T) {
	// The controller builds ClientConfig without any timeout fields; the
	// client must default them before use or every operation times out at 0s.
	config := &ClientConfig{Host: "localhost", Port: 1883, ClientID: testClientID}
	logger := slog.New(slog.DiscardHandler)

	client, err := NewClient(config, logger)
	if err != nil {
		t.Fatalf("NewClient() failed: %v", err)
	}
	pahoClient, ok := client.(*PahoClient)
	if !ok {
		t.Fatalf("NewClient() returned %T, want *PahoClient", client)
	}

	if pahoClient.config.RequestTimeout != DefaultRequestTimeout {
		t.Errorf("RequestTimeout = %v, want %v", pahoClient.config.RequestTimeout, DefaultRequestTimeout)
	}
	if pahoClient.config.ConnectTimeout != 10*time.Second {
		t.Errorf("ConnectTimeout = %v, want 10s", pahoClient.config.ConnectTimeout)
	}
	if pahoClient.config.KeepAlive != 60*time.Second {
		t.Errorf("KeepAlive = %v, want 60s", pahoClient.config.KeepAlive)
	}
}

func TestStatePayloadJSON(t *testing.T) {
	payload := StatePayload{
		Reading:  123.45,
		LastSeen: testLastSeen,
	}

	data, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("Failed to marshal StatePayload: %v", err)
	}

	var decoded StatePayload
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal StatePayload: %v", err)
	}

	if decoded.LastSeen != payload.LastSeen {
		t.Errorf("Expected LastSeen %s, got %s", payload.LastSeen, decoded.LastSeen)
	}

	// Reading is interface{}, check it can be decoded
	if decoded.Reading == nil {
		t.Error("Reading should not be nil after unmarshal")
	}
}

func TestHADiscoveryPayloadJSON(t *testing.T) {
	payload := HADiscoveryPayload{
		Device: DeviceInfo{
			Identifiers:  testIdentifiers,
			Name:         testMeterName,
			Manufacturer: "Test Mfg",
			Model:        "Model X",
			SWVersion:    testSWVersion,
			SerialNumber: "12345",
			HWVersion:    testHWVersion,
		},
		Origin: OriginInfo{
			Name:       testOriginName,
			SWVersion:  testSWVersion,
			SupportURL: testSupportURL,
		},
		Components: map[string]Component{
			"test_reading": {
				Platform:      testPlatform,
				Name:          "Reading",
				ValueTemplate: "{{ value_json.reading }}",
				UniqueID:      "test_reading",
				DeviceClass:   testDeviceClass,
				StateClass:    testStateClass,
			},
		},
		StateTopic:        "rtlamr/12345/state",
		AvailabilityTopic: "rtlamr/status",
		QoS:               1,
	}

	data, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("Failed to marshal HADiscoveryPayload: %v", err)
	}

	var decoded HADiscoveryPayload
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal HADiscoveryPayload: %v", err)
	}

	if decoded.Device.Name != payload.Device.Name {
		t.Errorf("Expected device name %s, got %s", payload.Device.Name, decoded.Device.Name)
	}

	if decoded.Origin.Name != payload.Origin.Name {
		t.Errorf("Expected origin name %s, got %s", payload.Origin.Name, decoded.Origin.Name)
	}

	if len(decoded.Components) != len(payload.Components) {
		t.Errorf("Expected %d components, got %d", len(payload.Components), len(decoded.Components))
	}

	if decoded.QoS != payload.QoS {
		t.Errorf("Expected QoS %d, got %d", payload.QoS, decoded.QoS)
	}
}

func TestDeviceInfo(t *testing.T) {
	device := DeviceInfo{
		Identifiers:  testIdentifiers,
		Name:         testMeterName,
		Manufacturer: "Test Manufacturer",
		Model:        "Test Model",
		SWVersion:    testSWVersion,
		SerialNumber: "SN12345",
		HWVersion:    testHWVersion,
	}

	if device.Identifiers != testIdentifiers {
		t.Errorf("Expected identifiers 'meter_12345', got '%s'", device.Identifiers)
	}
	if device.Name != testMeterName {
		t.Errorf("Expected name 'Test Meter', got '%s'", device.Name)
	}
	if device.Manufacturer != "Test Manufacturer" {
		t.Errorf("Expected manufacturer 'Test Manufacturer', got '%s'", device.Manufacturer)
	}
	if device.Model != "Test Model" {
		t.Errorf("Expected model 'Test Model', got '%s'", device.Model)
	}
	if device.SWVersion != testSWVersion {
		t.Errorf("Expected sw_version '1.0.0', got '%s'", device.SWVersion)
	}
	if device.SerialNumber != "SN12345" {
		t.Errorf("Expected serial_number 'SN12345', got '%s'", device.SerialNumber)
	}
	if device.HWVersion != testHWVersion {
		t.Errorf("Expected hw_version '1.0', got '%s'", device.HWVersion)
	}
}

func TestOriginInfo(t *testing.T) {
	origin := OriginInfo{
		Name:       testOriginName,
		SWVersion:  testSWVersion,
		SupportURL: testSupportURL,
	}

	if origin.Name != testOriginName {
		t.Errorf("Expected name 'rtlsdr2mqtt', got '%s'", origin.Name)
	}
	if origin.SWVersion != testSWVersion {
		t.Errorf("Expected version '1.0.0', got '%s'", origin.SWVersion)
	}
	if origin.SupportURL != testSupportURL {
		t.Errorf("Expected support_url 'https://github.com/test/test', got '%s'", origin.SupportURL)
	}
}

func TestComponent(t *testing.T) {
	component := Component{
		Platform:            testPlatform,
		Name:                "Test Sensor",
		ValueTemplate:       "{{ value_json.value }}",
		UniqueID:            "test_12345",
		JSONAttributesTopic: "test/attributes",
		DeviceClass:         testDeviceClass,
		StateClass:          testStateClass,
		UnitOfMeasurement:   "kWh",
		Icon:                "mdi:flash",
		ExpireAfter:         3600,
		ForceUpdate:         true,
	}

	if component.Platform != testPlatform {
		t.Errorf("Expected platform 'sensor', got '%s'", component.Platform)
	}
	if component.Name != "Test Sensor" {
		t.Errorf("Expected name 'Test Sensor', got '%s'", component.Name)
	}
	if component.ValueTemplate != "{{ value_json.value }}" {
		t.Errorf("Expected value_template '{{ value_json.value }}', got '%s'", component.ValueTemplate)
	}
	if component.UniqueID != "test_12345" {
		t.Errorf("Expected unique_id 'test_12345', got '%s'", component.UniqueID)
	}
	if component.JSONAttributesTopic != "test/attributes" {
		t.Errorf("Expected json_attributes_topic 'test/attributes', got '%s'", component.JSONAttributesTopic)
	}
	if component.DeviceClass != testDeviceClass {
		t.Errorf("Expected device class 'energy', got '%s'", component.DeviceClass)
	}
	if component.StateClass != testStateClass {
		t.Errorf("Expected state_class 'total_increasing', got '%s'", component.StateClass)
	}
	if component.UnitOfMeasurement != "kWh" {
		t.Errorf("Expected unit_of_measurement 'kWh', got '%s'", component.UnitOfMeasurement)
	}
	if component.Icon != "mdi:flash" {
		t.Errorf("Expected icon 'mdi:flash', got '%s'", component.Icon)
	}
	if component.ExpireAfter != 3600 {
		t.Errorf("Expected expire after 3600, got %d", component.ExpireAfter)
	}
	if !component.ForceUpdate {
		t.Error("Expected force update to be true")
	}
}

func TestClientConfig(t *testing.T) {
	cfg := ClientConfig{
		Host:           "mqtt.example.com",
		Port:           1883,
		Username:       "user",
		Password:       "pass",
		ClientID:       "test-client",
		TLSEnabled:     true,
		TLSInsecure:    false,
		TLSCA:          "/path/to/ca.crt",
		TLSCert:        "/path/to/cert.pem",
		TLSKey:         "/path/to/key.pem",
		KeepAlive:      60 * time.Second,
		ConnectTimeout: 10 * time.Second,
		WillTopic:      "test/status",
		WillPayload:    "offline",
		WillQoS:        1,
		WillRetain:     true,
	}

	t.Run("connection_settings", func(t *testing.T) {
		assertConnectionSettings(t, &cfg)
	})

	t.Run("tls_settings", func(t *testing.T) {
		assertTLSSettings(t, &cfg)
	})

	t.Run("will_settings", func(t *testing.T) {
		assertWillSettings(t, &cfg)
	})
}

func assertConnectionSettings(t *testing.T, cfg *ClientConfig) {
	t.Helper()
	if cfg.Host != "mqtt.example.com" {
		t.Errorf("Expected host 'mqtt.example.com', got '%s'", cfg.Host)
	}
	if cfg.Port != 1883 {
		t.Errorf("Expected port 1883, got %d", cfg.Port)
	}
	if cfg.Username != "user" {
		t.Errorf("Expected username 'user', got '%s'", cfg.Username)
	}
	if cfg.Password != "pass" {
		t.Errorf("Expected password 'pass', got '%s'", cfg.Password)
	}
	if cfg.ClientID != "test-client" {
		t.Errorf("Expected client_id 'test-client', got '%s'", cfg.ClientID)
	}
	if cfg.KeepAlive != 60*time.Second {
		t.Errorf("Expected keep alive 60s, got %v", cfg.KeepAlive)
	}
	if cfg.ConnectTimeout != 10*time.Second {
		t.Errorf("Expected connect_timeout 10s, got %v", cfg.ConnectTimeout)
	}
}

func assertTLSSettings(t *testing.T, cfg *ClientConfig) {
	t.Helper()
	if !cfg.TLSEnabled {
		t.Error("Expected TLS enabled")
	}
	if cfg.TLSInsecure {
		t.Error("Expected TLS secure")
	}
	if cfg.TLSCA != "/path/to/ca.crt" {
		t.Errorf("Expected tls_ca '/path/to/ca.crt', got '%s'", cfg.TLSCA)
	}
	if cfg.TLSCert != "/path/to/cert.pem" {
		t.Errorf("Expected tls_cert '/path/to/cert.pem', got '%s'", cfg.TLSCert)
	}
	if cfg.TLSKey != "/path/to/key.pem" {
		t.Errorf("Expected tls_key '/path/to/key.pem', got '%s'", cfg.TLSKey)
	}
}

func assertWillSettings(t *testing.T, cfg *ClientConfig) {
	t.Helper()
	if cfg.WillTopic != "test/status" {
		t.Errorf("Expected will_topic 'test/status', got '%s'", cfg.WillTopic)
	}
	if cfg.WillPayload != "offline" {
		t.Errorf("Expected will_payload 'offline', got '%s'", cfg.WillPayload)
	}
	if cfg.WillQoS != 1 {
		t.Errorf("Expected will_qos 1, got %d", cfg.WillQoS)
	}
	if !cfg.WillRetain {
		t.Error("Expected will retain to be true")
	}
}

func TestComponentJSONOmitEmpty(t *testing.T) {
	// Test that omitempty works correctly
	component := Component{
		Platform:      testPlatform,
		Name:          "Test",
		ValueTemplate: "{{ value }}",
		UniqueID:      "test",
		// All optional fields left empty
	}

	data, err := json.Marshal(component)
	if err != nil {
		t.Fatalf("Failed to marshal component: %v", err)
	}

	jsonStr := string(data)

	// Optional fields should not appear in JSON
	if containsKey(jsonStr, "json_attributes_topic") {
		t.Error("json_attributes_topic should be omitted when empty")
	}

	if containsKey(jsonStr, "device_class") {
		t.Error("device_class should be omitted when empty")
	}

	if containsKey(jsonStr, "state_class") {
		t.Error("state_class should be omitted when empty")
	}

	if containsKey(jsonStr, "unit_of_measurement") {
		t.Error("unit_of_measurement should be omitted when empty")
	}

	if containsKey(jsonStr, "icon") {
		t.Error("icon should be omitted when empty")
	}

	if containsKey(jsonStr, "expire_after") {
		t.Error("expire_after should be omitted when zero")
	}

	if containsKey(jsonStr, "force_update") {
		t.Error("force_update should be omitted when false")
	}
}

func TestStatePayloadWithIntegerReading(t *testing.T) {
	payload := StatePayload{
		Reading:  12345,
		LastSeen: testLastSeen,
	}

	data, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var decoded StatePayload
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	// Reading should be decoded as float64 (JSON number default)
	if reading, ok := decoded.Reading.(float64); !ok || reading != 12345 {
		t.Errorf("Expected reading 12345, got %v (%T)", decoded.Reading, decoded.Reading)
	}
}

func TestStatePayloadWithFloatReading(t *testing.T) {
	payload := StatePayload{
		Reading:  123.456,
		LastSeen: testLastSeen,
	}

	data, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var decoded StatePayload
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if reading, ok := decoded.Reading.(float64); !ok || reading != 123.456 {
		t.Errorf("Expected reading 123.456, got %v (%T)", decoded.Reading, decoded.Reading)
	}
}

func TestStatePayloadWithStringReading(t *testing.T) {
	payload := StatePayload{
		Reading:  "unavailable",
		LastSeen: testLastSeen,
	}

	data, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var decoded StatePayload
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if reading, ok := decoded.Reading.(string); !ok || reading != "unavailable" {
		t.Errorf("Expected reading 'unavailable', got %v (%T)", decoded.Reading, decoded.Reading)
	}
}

// Helper function to check if JSON string contains a key.
func containsKey(jsonStr, key string) bool {
	var m map[string]any
	if err := json.Unmarshal([]byte(jsonStr), &m); err != nil {
		return false
	}
	_, exists := m[key]
	return exists
}
