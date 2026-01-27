package decoder

import (
	"testing"

	"github.com/bemasher/rtlamr/idm"
	"github.com/bemasher/rtlamr/netidm"
	"github.com/bemasher/rtlamr/r900"
	"github.com/bemasher/rtlamr/r900bcd"
	"github.com/bemasher/rtlamr/scm"
	"github.com/bemasher/rtlamr/scmplus"
)

func TestExtractMessageDataSCM(t *testing.T) {
	msg := scm.SCM{
		ID:          12345678,
		Type:        7,
		TamperPhy:   0,
		TamperEnc:   0,
		Consumption: 123456,
		ChecksumVal: 0x1234,
	}

	consumption, attrs := extractMessageData(msg)

	if consumption != 123456 {
		t.Errorf("Expected consumption 123456, got %d", consumption)
	}

	if attrs["id"] != uint32(12345678) {
		t.Errorf("Expected ID 12345678, got %v", attrs["id"])
	}

	if attrs["type"] != uint8(7) {
		t.Errorf("Expected type 7, got %v", attrs["type"])
	}

	if attrs["tamper_phy"] != uint8(0) {
		t.Errorf("Expected tamper_phy 0, got %v", attrs["tamper_phy"])
	}

	if attrs["tamper_enc"] != uint8(0) {
		t.Errorf("Expected tamper_enc 0, got %v", attrs["tamper_enc"])
	}

	if attrs["checksum"] != "0x1234" {
		t.Errorf("Expected checksum 0x1234, got %v", attrs["checksum"])
	}

	if attrs["protocol"] != "SCM" {
		t.Errorf("Expected protocol SCM, got %v", attrs["protocol"])
	}

	if attrs["meter_type"] != uint8(7) {
		t.Errorf("Expected meter_type 7, got %v", attrs["meter_type"])
	}
}

func TestExtractMessageDataSCMPlus(t *testing.T) {
	msg := scmplus.SCM{
		EndpointID:   87654321,
		EndpointType: 5,
		ProtocolID:   4,
		Tamper:       1,
		Consumption:  999999,
		PacketCRC:    0xABCD,
	}

	consumption, attrs := extractMessageData(msg)

	if consumption != 999999 {
		t.Errorf("Expected consumption 999999, got %d", consumption)
	}

	if attrs["endpoint_id"] != uint32(87654321) {
		t.Errorf("Expected endpoint_id 87654321, got %v", attrs["endpoint_id"])
	}

	if attrs["endpoint_type"] != uint8(5) {
		t.Errorf("Expected endpoint_type 5, got %v", attrs["endpoint_type"])
	}

	if attrs["protocol_id"] != uint8(4) {
		t.Errorf("Expected protocol_id 4, got %v", attrs["protocol_id"])
	}

	if attrs["tamper"] != uint16(1) {
		t.Errorf("Expected tamper 1, got %v", attrs["tamper"])
	}

	if attrs["checksum"] != "0xABCD" {
		t.Errorf("Expected checksum 0xABCD, got %v", attrs["checksum"])
	}
}

func TestExtractMessageDataR900(t *testing.T) {
	msg := r900.R900{
		ID:          11223344,
		Unkn1:       1,
		NoUse:       0,
		BackFlow:    0,
		Consumption: 555555,
		Unkn3:       2,
		Leak:        0,
		LeakNow:     0,
	}

	consumption, attrs := extractMessageData(msg)

	if consumption != 555555 {
		t.Errorf("Expected consumption 555555, got %d", consumption)
	}

	if attrs["id"] != uint32(11223344) {
		t.Errorf("Expected id 11223344, got %v", attrs["id"])
	}

	if attrs["unkn1"] != uint8(1) {
		t.Errorf("Expected unkn1 1, got %v", attrs["unkn1"])
	}

	if attrs["no_use"] != uint8(0) {
		t.Errorf("Expected no_use 0, got %v", attrs["no_use"])
	}

	if attrs["back_flow"] != uint8(0) {
		t.Errorf("Expected back_flow 0, got %v", attrs["back_flow"])
	}

	if attrs["unkn3"] != uint8(2) {
		t.Errorf("Expected unkn3 2, got %v", attrs["unkn3"])
	}

	if attrs["leak"] != uint8(0) {
		t.Errorf("Expected leak 0, got %v", attrs["leak"])
	}

	if attrs["leak_now"] != uint8(0) {
		t.Errorf("Expected leak_now 0, got %v", attrs["leak_now"])
	}
}

func TestExtractMessageDataR900BCD(t *testing.T) {
	msg := r900bcd.R900BCD{
		R900: r900.R900{
			ID:          99887766,
			Unkn1:       1,
			NoUse:       0,
			BackFlow:    1,
			Consumption: 777777,
			Unkn3:       3,
			Leak:        1,
			LeakNow:     0,
		},
	}

	consumption, attrs := extractMessageData(msg)

	if consumption != 777777 {
		t.Errorf("Expected consumption 777777, got %d", consumption)
	}

	if attrs["id"] != uint32(99887766) {
		t.Errorf("Expected id 99887766, got %v", attrs["id"])
	}

	if attrs["back_flow"] != uint8(1) {
		t.Errorf("Expected back_flow 1, got %v", attrs["back_flow"])
	}

	if attrs["leak"] != uint8(1) {
		t.Errorf("Expected leak 1, got %v", attrs["leak"])
	}
}

func TestExtractMessageDataIDM(t *testing.T) {
	msg := idm.IDM{
		ERTSerialNumber:          12345678,
		ERTType:                  5,
		ConsumptionIntervalCount: 100,
		ModuleProgrammingState:   1,
		TamperCounters:           []byte{1, 2, 3, 4, 5, 6},
		AsynchronousCounters:     10,
		PowerOutageFlags:         []byte{0, 0, 0, 0, 0, 0},
		LastConsumptionCount:     888888,
		TransmitTimeOffset:       30,
		PacketCRC:                0x5678,
	}

	consumption, attrs := extractMessageData(msg)

	if consumption != 888888 {
		t.Errorf("Expected consumption 888888, got %d", consumption)
	}

	if attrs["ert_serial_number"] != uint32(12345678) {
		t.Errorf("Expected ert_serial_number 12345678, got %v", attrs["ert_serial_number"])
	}

	if attrs["ert_type"] != uint8(5) {
		t.Errorf("Expected ert_type 5, got %v", attrs["ert_type"])
	}

	if attrs["consumption_interval_count"] != uint8(100) {
		t.Errorf("Expected consumption_interval_count 100, got %v", attrs["consumption_interval_count"])
	}

	if attrs["module_programming_state"] != uint8(1) {
		t.Errorf("Expected module_programming_state 1, got %v", attrs["module_programming_state"])
	}

	if attrs["asynchronous_counters"] != uint16(10) {
		t.Errorf("Expected asynchronous_counters 10, got %v", attrs["asynchronous_counters"])
	}

	if attrs["transmit_time_offset"] != uint16(30) {
		t.Errorf("Expected transmit_time_offset 30, got %v", attrs["transmit_time_offset"])
	}

	if attrs["checksum"] != "0x5678" {
		t.Errorf("Expected checksum 0x5678, got %v", attrs["checksum"])
	}

	// Check tamper_counters is formatted as hex string
	if _, ok := attrs["tamper_counters"].(string); !ok {
		t.Errorf("Expected tamper_counters to be a string, got %T", attrs["tamper_counters"])
	}

	// Check power_outage_flags is formatted as hex string
	if _, ok := attrs["power_outage_flags"].(string); !ok {
		t.Errorf("Expected power_outage_flags to be a string, got %T", attrs["power_outage_flags"])
	}
}

func TestExtractMessageDataNetIDM(t *testing.T) {
	msg := netidm.NetIDM{
		ERTSerialNumber:          11223344,
		ERTType:                  7,
		ConsumptionIntervalCount: 150,
		ProgrammingState:         2,
		LastGeneration:           100,
		LastConsumption:          666666,
		LastConsumptionNet:       666600,
		TransmitTimeOffset:       45,
		PacketCRC:                0x9ABC,
	}

	consumption, attrs := extractMessageData(msg)

	if consumption != 666666 {
		t.Errorf("Expected consumption 666666, got %d", consumption)
	}

	if attrs["ert_serial_number"] != uint32(11223344) {
		t.Errorf("Expected ert_serial_number 11223344, got %v", attrs["ert_serial_number"])
	}

	if attrs["ert_type"] != uint8(7) {
		t.Errorf("Expected ert_type 7, got %v", attrs["ert_type"])
	}

	if attrs["consumption_interval_count"] != uint8(150) {
		t.Errorf("Expected consumption_interval_count 150, got %v", attrs["consumption_interval_count"])
	}

	if attrs["programming_state"] != uint8(2) {
		t.Errorf("Expected programming_state 2, got %v", attrs["programming_state"])
	}

	if attrs["last_generation"] != uint32(100) {
		t.Errorf("Expected last_generation 100, got %v", attrs["last_generation"])
	}

	if attrs["last_consumption"] != uint32(666666) {
		t.Errorf("Expected last_consumption 666666, got %v", attrs["last_consumption"])
	}

	if attrs["last_consumption_net"] != uint32(666600) {
		t.Errorf("Expected last_consumption_net 666600, got %v", attrs["last_consumption_net"])
	}

	if attrs["transmit_time_offset"] != uint16(45) {
		t.Errorf("Expected transmit_time_offset 45, got %v", attrs["transmit_time_offset"])
	}

	if attrs["checksum"] != "0x9ABC" {
		t.Errorf("Expected checksum 0x9ABC, got %v", attrs["checksum"])
	}
}

func TestExtractMessageDataUnknownType(t *testing.T) {
	// Test with mock message that doesn't match any known type
	msg := mockMessage{meterID: 12345, meterType: 7, msgType: "UNKNOWN"}

	consumption, attrs := extractMessageData(msg)

	if consumption != 0 {
		t.Errorf("Expected consumption 0 for unknown type, got %d", consumption)
	}

	if attrs["protocol"] != "UNKNOWN" {
		t.Errorf("Expected protocol UNKNOWN, got %v", attrs["protocol"])
	}

	if attrs["meter_type"] != uint8(7) {
		t.Errorf("Expected meter_type 7, got %v", attrs["meter_type"])
	}
}

func TestMessageMeterIDString(t *testing.T) {
	msg := &Message{
		MeterID: 12345678,
	}

	result := msg.MeterIDString()
	expected := "12345678"

	if result != expected {
		t.Errorf("Expected MeterIDString() = %s, got %s", expected, result)
	}
}

func TestMessageConsumptionInt64(t *testing.T) {
	msg := &Message{
		Consumption: 999999,
	}

	result := msg.ConsumptionInt64()
	expected := int64(999999)

	if result != expected {
		t.Errorf("Expected ConsumptionInt64() = %d, got %d", expected, result)
	}
}
