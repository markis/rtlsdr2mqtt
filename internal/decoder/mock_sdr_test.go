package decoder

import (
	"sync"

	"rtlsdr2mqtt/internal/sdr"
)

// mockSDR implements sdr.SDR for testing the decoder without hardware.
type mockSDR struct {
	mu sync.Mutex

	opened      bool
	bufferReset bool
	streaming   bool

	streamChan   chan []byte
	startBufLen  uint32
	startBufNum  uint32
	stopErr      error
	closeErr     error
	streamBlocks [][]byte
}

func (m *mockSDR) Open() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.opened = true
	return nil
}

func (m *mockSDR) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.opened = false
	return m.closeErr
}

func (m *mockSDR) SetCenterFreq(uint32) error {
	return nil
}

func (m *mockSDR) SetSampleRate(uint32) error {
	return nil
}

func (m *mockSDR) SetGainMode(bool) error {
	return nil
}

func (m *mockSDR) SetGain(int) error {
	return nil
}

func (m *mockSDR) SetAGCMode(bool) error {
	return nil
}

func (m *mockSDR) SetFreqCorrection(int) error {
	return nil
}

func (m *mockSDR) ResetBuffer() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.bufferReset = true
	return nil
}

func (m *mockSDR) StartStreaming(bufLen, bufNum uint32) (<-chan []byte, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.startBufLen = bufLen
	m.startBufNum = bufNum
	if m.streaming {
		return nil, sdr.ErrAlreadyStreaming
	}
	m.streaming = true
	m.streamChan = make(chan []byte, len(m.streamBlocks)+2)
	return m.streamChan, nil
}

func (m *mockSDR) StopStreaming() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if !m.streaming {
		return nil
	}
	m.streaming = false
	close(m.streamChan)
	return m.stopErr
}

func (m *mockSDR) GetDeviceCount() uint32 {
	return 1
}

func (m *mockSDR) GetDeviceName(uint32) string {
	return "Mock RTL2832"
}

func (m *mockSDR) GetTunerGains() []int {
	return []int{0, 100, 200}
}

// push delivers a sample block to the stream (non-blocking; the decoder drops on overflow).
func (m *mockSDR) push(block []byte) {
	m.mu.Lock()
	ch := m.streamChan
	m.mu.Unlock()
	if ch == nil {
		return
	}
	select {
	case ch <- block:
	default:
	}
}

func (m *mockSDR) isOpen() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.opened
}

func (m *mockSDR) isStreaming() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.streaming
}

var _ sdr.SDR = (*mockSDR)(nil)
