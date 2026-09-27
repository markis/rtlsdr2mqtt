package decoder

import (
	"testing"

	"github.com/bemasher/rtlamr/protocol"

	"rtlsdr2mqtt/internal/config"
)

// fakeMessage implements protocol.Message for dedup tests without RF data.
type fakeMessage struct {
	msgType   string
	meterID   uint32
	meterType uint8
	checksum  []byte
}

func (m fakeMessage) Record() []string { return nil }
func (m fakeMessage) MsgType() string  { return m.msgType }
func (m fakeMessage) MeterID() uint32  { return m.meterID }
func (m fakeMessage) MeterType() uint8 { return m.meterType }
func (m fakeMessage) Checksum() []byte { return m.checksum }

var _ protocol.Message = fakeMessage{}

// TestDecodeBlockDedupAcrossBlocks guards the cross-block digest swap: a
// message seen in two consecutive blocks is forwarded once, and a new
// message in a later block is forwarded again.
func TestDecodeBlockDedupAcrossBlocks(t *testing.T) {
	cfg := &config.Config{
		SDR: config.SDRConfig{USBDevice: ""},
		Meters: []config.MeterConfig{
			{ID: testMeterID, Protocol: testProtocolSCM},
		},
	}
	d := NewDecoder(cfg, newTestLogger())

	prev := make(map[protocol.Digest]bool)
	next := make(map[protocol.Digest]bool)

	msg := fakeMessage{msgType: "scm+", meterID: 12345678, meterType: 4, checksum: []byte{0xAB}}

	// First block: new message is forwarded.
	if !d.handleDecodedMessage(msg, prev, next) {
		t.Fatal("expected first sighting to be forwarded")
	}
	prev, next = next, prev
	for key := range next {
		delete(next, key)
	}

	// Second block, same message: deduplicated.
	if d.handleDecodedMessage(msg, prev, next) {
		t.Fatal("expected repeated message to be deduplicated")
	}
	prev, next = next, prev
	for key := range next {
		delete(next, key)
	}

	// Third block, same message: still deduplicated.
	if d.handleDecodedMessage(msg, prev, next) {
		t.Fatal("expected repeated message to stay deduplicated")
	}

	// A different message is forwarded.
	other := fakeMessage{msgType: "scm+", meterID: 87654321, meterType: 4, checksum: []byte{0xCD}}
	prev, next = next, prev
	for key := range next {
		delete(next, key)
	}
	if !d.handleDecodedMessage(other, prev, next) {
		t.Fatal("expected new message to be forwarded")
	}

	select {
	case got := <-d.msgChan:
		if got.MeterIDString() != "12345678" {
			t.Errorf("expected first forwarded meter 12345678, got %s", got.MeterIDString())
		}
	default:
		t.Fatal("expected a forwarded message on msgChan")
	}
	select {
	case got := <-d.msgChan:
		if got.MeterIDString() != "87654321" {
			t.Errorf("expected second forwarded meter 87654321, got %s", got.MeterIDString())
		}
	default:
		t.Fatal("expected the new message on msgChan")
	}
}
