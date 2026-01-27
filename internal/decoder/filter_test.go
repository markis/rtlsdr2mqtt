package decoder

import (
	"testing"

	_ "github.com/bemasher/rtlamr/protocol"
)

// mockMessage is a simple mock implementation of protocol.Message for testing.
type mockMessage struct {
	meterID   uint32
	meterType uint8
	msgType   string
}

func (m mockMessage) MeterID() uint32 {
	return m.meterID
}

func (m mockMessage) MeterType() uint8 {
	return m.meterType
}

func (m mockMessage) MsgType() string {
	return m.msgType
}

func (m mockMessage) Checksum() []byte {
	return []byte{}
}

func (m mockMessage) Record() []string {
	return []string{}
}

func TestFilterChainAdd(t *testing.T) {
	fc := filterChain{}
	filter := newMeterIDFilter([]uint32{12345})

	if len(fc) != 0 {
		t.Errorf("Expected empty filter chain, got length %d", len(fc))
	}

	fc.add(filter)

	if len(fc) != 1 {
		t.Errorf("Expected filter chain length 1, got %d", len(fc))
	}
}

func TestFilterChainMatchEmpty(t *testing.T) {
	fc := filterChain{}
	msg := mockMessage{meterID: 12345}

	// Empty filter chain should match everything
	if !fc.match(msg) {
		t.Error("Empty filter chain should match all messages")
	}
}

func TestFilterChainMatchSingleFilter(t *testing.T) {
	fc := filterChain{}
	filter := newMeterIDFilter([]uint32{12345, 67890})
	fc.add(filter)

	tests := []struct {
		meterID uint32
		matches bool
	}{
		{12345, true},
		{67890, true},
		{99999, false},
		{11111, false},
	}

	for _, tt := range tests {
		msg := mockMessage{meterID: tt.meterID}
		result := fc.match(msg)

		if result != tt.matches {
			t.Errorf("meterID %d: expected match=%v, got %v", tt.meterID, tt.matches, result)
		}
	}
}

func TestFilterChainMatchMultipleFilters(t *testing.T) {
	fc := filterChain{}
	filter1 := newMeterIDFilter([]uint32{12345, 67890})
	filter2 := newMeterIDFilter([]uint32{12345, 11111}) // Only 12345 is in both
	fc.add(filter1)
	fc.add(filter2)

	tests := []struct {
		meterID uint32
		matches bool
	}{
		{12345, true},  // In both filters
		{67890, false}, // Only in filter1
		{11111, false}, // Only in filter2
		{99999, false}, // In neither
	}

	for _, tt := range tests {
		msg := mockMessage{meterID: tt.meterID}
		result := fc.match(msg)

		if result != tt.matches {
			t.Errorf("meterID %d: expected match=%v, got %v", tt.meterID, tt.matches, result)
		}
	}
}

func TestNewMeterIDFilter(t *testing.T) {
	ids := []uint32{12345, 67890, 11111}
	filter := newMeterIDFilter(ids)

	if filter == nil {
		t.Fatal("newMeterIDFilter returned nil")
	}

	if len(filter.ids) != 3 {
		t.Errorf("Expected 3 IDs in filter, got %d", len(filter.ids))
	}

	for _, id := range ids {
		if !filter.ids[id] {
			t.Errorf("Expected ID %d to be in filter", id)
		}
	}
}

func TestNewMeterIDFilterEmpty(t *testing.T) {
	filter := newMeterIDFilter([]uint32{})

	if filter == nil {
		t.Fatal("newMeterIDFilter returned nil")
	}

	if len(filter.ids) != 0 {
		t.Errorf("Expected 0 IDs in filter, got %d", len(filter.ids))
	}
}

func TestMeterIDFilterFilter(t *testing.T) {
	filter := newMeterIDFilter([]uint32{12345, 67890})

	tests := []struct {
		meterID uint32
		matches bool
	}{
		{12345, true},
		{67890, true},
		{11111, false},
		{99999, false},
		{0, false},
	}

	for _, tt := range tests {
		msg := mockMessage{meterID: tt.meterID}
		result := filter.filter(msg)

		if result != tt.matches {
			t.Errorf("meterID %d: expected match=%v, got %v", tt.meterID, tt.matches, result)
		}
	}
}

func TestMeterIDFilterEmptyList(t *testing.T) {
	filter := newMeterIDFilter([]uint32{})
	msg := mockMessage{meterID: 12345}

	// Empty filter should not match anything
	if filter.filter(msg) {
		t.Error("Empty filter should not match any message")
	}
}

func TestMeterIDFilterDuplicateIDs(t *testing.T) {
	// Test that duplicate IDs in the list don't cause issues
	filter := newMeterIDFilter([]uint32{12345, 12345, 67890, 67890})

	if len(filter.ids) != 2 {
		t.Errorf("Expected 2 unique IDs in filter, got %d", len(filter.ids))
	}

	msg1 := mockMessage{meterID: 12345}
	if !filter.filter(msg1) {
		t.Error("Expected meterID 12345 to match")
	}

	msg2 := mockMessage{meterID: 67890}
	if !filter.filter(msg2) {
		t.Error("Expected meterID 67890 to match")
	}
}
