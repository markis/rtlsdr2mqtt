package parser

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

func TestNewParser(t *testing.T) {
	meterIDs := []string{"12345", "67890"}
	parser := NewParser(meterIDs)

	if parser == nil {
		t.Fatal("NewParser returned nil")
	}

	if len(parser.meterIDs) != 2 {
		t.Errorf("Expected 2 meter IDs, got %d", len(parser.meterIDs))
	}

	if !parser.meterIDs["12345"] {
		t.Error("Expected meter ID 12345 to be in map")
	}

	if !parser.meterIDs["67890"] {
		t.Error("Expected meter ID 67890 to be in map")
	}
}

func TestParseLineEmptyString(t *testing.T) {
	parser := NewParser([]string{"12345"})
	msg, err := parser.ParseLine("")

	if msg != nil {
		t.Error("Expected nil message for empty string")
	}

	if !errors.Is(err, ErrIgnoredLine) {
		t.Errorf("Expected ErrIgnoredLine, got %v", err)
	}
}

func TestParseLineWhitespace(t *testing.T) {
	parser := NewParser([]string{"12345"})
	msg, err := parser.ParseLine("   \t\n  ")

	if msg != nil {
		t.Error("Expected nil message for whitespace")
	}

	if !errors.Is(err, ErrIgnoredLine) {
		t.Errorf("Expected ErrIgnoredLine, got %v", err)
	}
}

func TestParseLineNonJSON(t *testing.T) {
	parser := NewParser([]string{"12345"})
	msg, err := parser.ParseLine("This is not JSON")

	if msg != nil {
		t.Error("Expected nil message for non-JSON")
	}

	if !errors.Is(err, ErrIgnoredLine) {
		t.Errorf("Expected ErrIgnoredLine, got %v", err)
	}
}

func TestParseLineInvalidJSON(t *testing.T) {
	parser := NewParser([]string{"12345"})
	msg, err := parser.ParseLine("{invalid json")

	if msg != nil {
		t.Error("Expected nil message for invalid JSON")
	}

	if err == nil {
		t.Error("Expected error for invalid JSON")
	}
}

func TestParseLineSCMMessage(t *testing.T) {
	parser := NewParser([]string{"12345678"})

	scmMsg := SCMMessage{
		ID:          12345678,
		Type:        7,
		Consumption: 123456,
	}
	msgBytes, err := json.Marshal(scmMsg)
	if err != nil {
		t.Fatalf("Failed to marshal SCM message: %v", err)
	}

	rtlamrOutput := RTLAMROutput{
		Time:    "2024-01-01T12:00:00Z",
		Type:    "SCM",
		Message: msgBytes,
	}
	outputBytes, err := json.Marshal(rtlamrOutput)
	if err != nil {
		t.Fatalf("Failed to marshal RTLAMR output: %v", err)
	}

	parsed, err := parser.ParseLine(string(outputBytes))
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if parsed.MeterID != "12345678" {
		t.Errorf("Expected meter ID 12345678, got %s", parsed.MeterID)
	}

	if parsed.Consumption != 123456 {
		t.Errorf("Expected consumption 123456, got %d", parsed.Consumption)
	}

	if parsed.Protocol != "scm" {
		t.Errorf("Expected protocol 'scm', got %s", parsed.Protocol)
	}
}

func TestParseLineSCMPlusMessage(t *testing.T) {
	parser := NewParser([]string{"87654321"})

	scmMsg := SCMMessage{
		EndpointID:  87654321,
		Consumption: 999999,
	}
	msgBytes, err := json.Marshal(scmMsg)
	if err != nil {
		t.Fatalf("Failed to marshal SCM message: %v", err)
	}

	rtlamrOutput := RTLAMROutput{
		Time:    "2024-01-01T12:00:00Z",
		Type:    "SCM+",
		Message: msgBytes,
	}
	outputBytes, err := json.Marshal(rtlamrOutput)
	if err != nil {
		t.Fatalf("Failed to marshal RTLAMR output: %v", err)
	}

	parsed, err := parser.ParseLine(string(outputBytes))
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if parsed.MeterID != "87654321" {
		t.Errorf("Expected meter ID 87654321, got %s", parsed.MeterID)
	}

	if parsed.Consumption != 999999 {
		t.Errorf("Expected consumption 999999, got %d", parsed.Consumption)
	}

	if parsed.Protocol != "scm+" {
		t.Errorf("Expected protocol 'scm+', got %s", parsed.Protocol)
	}
}

func TestParseLineR900Message(t *testing.T) {
	parser := NewParser([]string{"11223344"})

	r900Msg := R900Message{
		ID:          11223344,
		Consumption: 555555,
		Leak:        0,
		LeakNow:     0,
	}
	msgBytes, err := json.Marshal(r900Msg)
	if err != nil {
		t.Fatalf("Failed to marshal R900 message: %v", err)
	}

	rtlamrOutput := RTLAMROutput{
		Time:    "2024-01-01T12:00:00Z",
		Type:    "R900",
		Message: msgBytes,
	}
	outputBytes, err := json.Marshal(rtlamrOutput)
	if err != nil {
		t.Fatalf("Failed to marshal RTLAMR output: %v", err)
	}

	parsed, err := parser.ParseLine(string(outputBytes))
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if parsed.MeterID != "11223344" {
		t.Errorf("Expected meter ID 11223344, got %s", parsed.MeterID)
	}

	if parsed.Consumption != 555555 {
		t.Errorf("Expected consumption 555555, got %d", parsed.Consumption)
	}

	if parsed.Protocol != "r900" {
		t.Errorf("Expected protocol 'r900', got %s", parsed.Protocol)
	}
}

func TestParseLineIDMMessage(t *testing.T) {
	parser := NewParser([]string{"99887766"})

	idmMsg := IDMMessage{
		ERTSerialNumber:      99887766,
		LastConsumptionCount: 777777,
	}
	msgBytes, err := json.Marshal(idmMsg)
	if err != nil {
		t.Fatalf("Failed to marshal IDM message: %v", err)
	}

	rtlamrOutput := RTLAMROutput{
		Time:    "2024-01-01T12:00:00Z",
		Type:    "IDM",
		Message: msgBytes,
	}
	outputBytes, err := json.Marshal(rtlamrOutput)
	if err != nil {
		t.Fatalf("Failed to marshal RTLAMR output: %v", err)
	}

	parsed, err := parser.ParseLine(string(outputBytes))
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if parsed.MeterID != "99887766" {
		t.Errorf("Expected meter ID 99887766, got %s", parsed.MeterID)
	}

	if parsed.Consumption != 777777 {
		t.Errorf("Expected consumption 777777, got %d", parsed.Consumption)
	}

	if parsed.Protocol != "idm" {
		t.Errorf("Expected protocol 'idm', got %s", parsed.Protocol)
	}
}

func TestParseLineFilteredMeterID(t *testing.T) {
	parser := NewParser([]string{"12345678"})

	scmMsg := SCMMessage{
		ID:          99999999, // Different meter ID
		Consumption: 123456,
	}
	msgBytes, err := json.Marshal(scmMsg)
	if err != nil {
		t.Fatalf("Failed to marshal SCM message: %v", err)
	}

	rtlamrOutput := RTLAMROutput{
		Time:    "2024-01-01T12:00:00Z",
		Type:    "SCM",
		Message: msgBytes,
	}
	outputBytes, err := json.Marshal(rtlamrOutput)
	if err != nil {
		t.Fatalf("Failed to marshal RTLAMR output: %v", err)
	}

	parsed, err := parser.ParseLine(string(outputBytes))

	if parsed != nil {
		t.Error("Expected nil message for filtered meter ID")
	}

	if !errors.Is(err, ErrIgnoredLine) {
		t.Errorf("Expected ErrIgnoredLine, got %v", err)
	}
}

func TestGetMessageForIDs(t *testing.T) {
	scmMsg := SCMMessage{
		ID:          12345678,
		Consumption: 123456,
	}
	msgBytes, err := json.Marshal(scmMsg)
	if err != nil {
		t.Fatalf("Failed to marshal SCM message: %v", err)
	}

	rtlamrOutput := RTLAMROutput{
		Time:    "2024-01-01T12:00:00Z",
		Type:    "SCM",
		Message: msgBytes,
	}
	outputBytes, err := json.Marshal(rtlamrOutput)
	if err != nil {
		t.Fatalf("Failed to marshal RTLAMR output: %v", err)
	}

	reading := GetMessageForIDs(string(outputBytes), []string{"12345678"})

	if reading == nil {
		t.Fatal("Expected non-nil reading")
	}

	if reading.MeterID != "12345678" {
		t.Errorf("Expected meter ID 12345678, got %s", reading.MeterID)
	}

	if reading.Consumption != 123456 {
		t.Errorf("Expected consumption 123456, got %d", reading.Consumption)
	}
}

func TestGetMessageForIDsNoMatch(t *testing.T) {
	scmMsg := SCMMessage{
		ID:          12345678,
		Consumption: 123456,
	}
	msgBytes, err := json.Marshal(scmMsg)
	if err != nil {
		t.Fatalf("Failed to marshal SCM message: %v", err)
	}

	rtlamrOutput := RTLAMROutput{
		Time:    "2024-01-01T12:00:00Z",
		Type:    "SCM",
		Message: msgBytes,
	}
	outputBytes, err := json.Marshal(rtlamrOutput)
	if err != nil {
		t.Fatalf("Failed to marshal RTLAMR output: %v", err)
	}

	reading := GetMessageForIDs(string(outputBytes), []string{"99999999"})

	if reading != nil {
		t.Error("Expected nil reading for non-matching meter ID")
	}
}

func TestGetMessageForIDsInvalidJSON(t *testing.T) {
	reading := GetMessageForIDs("invalid json", []string{"12345678"})

	if reading != nil {
		t.Error("Expected nil reading for invalid JSON")
	}
}

func TestFormatNumberNoFormat(t *testing.T) {
	result := FormatNumber(12345, "")

	if intResult, ok := result.(int64); !ok || intResult != 12345 {
		t.Errorf("Expected 12345, got %v", result)
	}
}

func TestFormatNumberNoDecimal(t *testing.T) {
	result := FormatNumber(12345, "######")

	if intResult, ok := result.(int64); !ok || intResult != 12345 {
		t.Errorf("Expected 12345, got %v", result)
	}
}

func TestFormatNumberWithDecimal(t *testing.T) {
	result := FormatNumber(12345, "###.##")

	floatResult, ok := result.(float64)
	if !ok {
		t.Fatalf("Expected float64, got %T", result)
	}

	if floatResult != 123.45 {
		t.Errorf("Expected 123.45, got %f", floatResult)
	}
}

func TestFormatNumberThreeDecimals(t *testing.T) {
	result := FormatNumber(123456, "###.###")

	floatResult, ok := result.(float64)
	if !ok {
		t.Fatalf("Expected float64, got %T", result)
	}

	if floatResult != 123.456 {
		t.Errorf("Expected 123.456, got %f", floatResult)
	}
}

func TestFormatNumberOneDecimal(t *testing.T) {
	result := FormatNumber(1234, "#.#")

	floatResult, ok := result.(float64)
	if !ok {
		t.Fatalf("Expected float64, got %T", result)
	}

	if floatResult != 123.4 {
		t.Errorf("Expected 123.4, got %f", floatResult)
	}
}

func TestFormatNumberZero(t *testing.T) {
	result := FormatNumber(0, "###.##")

	floatResult, ok := result.(float64)
	if !ok {
		t.Fatalf("Expected float64, got %T", result)
	}

	if floatResult != 0.0 {
		t.Errorf("Expected 0.0, got %f", floatResult)
	}
}

func TestRTLAMROutputGetMeterIDUnsupportedType(t *testing.T) {
	output := RTLAMROutput{
		Type:    "UNKNOWN",
		Message: json.RawMessage(`{}`),
	}

	_, err := output.GetMeterID()
	if err == nil {
		t.Error("Expected error for unsupported message type")
	}

	// Check if the error contains the unsupported message type error
	if !strings.Contains(err.Error(), "unsupported message type") {
		t.Errorf("Expected error to contain 'unsupported message type', got %v", err)
	}
}

func TestRTLAMROutputGetConsumptionUnsupportedType(t *testing.T) {
	output := RTLAMROutput{
		Type:    "UNKNOWN",
		Message: json.RawMessage(`{}`),
	}

	_, err := output.GetConsumption()
	if err == nil {
		t.Error("Expected error for unsupported message type")
	}

	// Check if the error contains the unsupported message type error
	if !strings.Contains(err.Error(), "unsupported message type") {
		t.Errorf("Expected error to contain 'unsupported message type', got %v", err)
	}
}

func TestRTLAMROutputGetAttributes(t *testing.T) {
	scmMsg := SCMMessage{
		ID:          12345678,
		Type:        7,
		Consumption: 123456,
		TamperPhy:   1,
		TamperEnc:   0,
	}
	msgBytes, err := json.Marshal(scmMsg)
	if err != nil {
		t.Fatalf("Failed to marshal SCM message: %v", err)
	}

	output := RTLAMROutput{
		Type:    "SCM",
		Message: msgBytes,
	}

	attrs, err := output.GetAttributes()
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if attrs["protocol"] != "scm" {
		t.Errorf("Expected protocol 'scm', got %v", attrs["protocol"])
	}

	if attrs["ID"] != float64(12345678) {
		t.Errorf("Expected ID 12345678, got %v", attrs["ID"])
	}
}

func TestRTLAMROutputGetAttributesInvalidJSON(t *testing.T) {
	output := RTLAMROutput{
		Type:    "SCM",
		Message: json.RawMessage(`{invalid`),
	}

	_, err := output.GetAttributes()
	if err == nil {
		t.Error("Expected error for invalid JSON")
	}
}
