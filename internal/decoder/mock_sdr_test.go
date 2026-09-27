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
	streamErrs   chan error
	startBufLen  uint32
	startBufNum  uint32
	stopErr      error
	closeErr     error
	streamBlocks [][]byte

	centerFreq uint32
	sampleRate uint32
	freqCalls  int
	rateCalls  int
	resetCalls int
	openCalls  int
	closeCalls int

	onClose func()
}

func (m *mockSDR) Open() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.openCalls++
	m.opened = true
	return nil
}

func (m *mockSDR) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.closeCalls++
	m.opened = false
	// Like the real device, closing a dead stream clears the streaming flag:
	// rtlsdr_close only runs after the async reader has exited.
	m.streaming = false
	if m.onClose != nil {
		m.onClose()
	}
	return m.closeErr
}

func (m *mockSDR) SetCenterFreq(freq uint32) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.freqCalls++
	m.centerFreq = freq
	return nil
}

func (m *mockSDR) SetSampleRate(rate uint32) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.rateCalls++
	m.sampleRate = rate
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
	m.resetCalls++
	return nil
}

func (m *mockSDR) StartStreaming(bufLen, bufNum uint32) (sdr.Stream, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.startBufLen = bufLen
	m.startBufNum = bufNum
	if m.streaming {
		return sdr.Stream{}, sdr.ErrAlreadyStreaming
	}
	m.streaming = true
	m.streamChan = make(chan []byte, len(m.streamBlocks)+2)
	m.streamErrs = make(chan error, 1)
	return sdr.Stream{Samples: m.streamChan, Errors: m.streamErrs}, nil
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

// failStream simulates the device dying out from under the decoder: the
// terminal error is delivered on the stream's Errors channel, streaming is
// marked dead, and the sample channel closes, mirroring librtlsdr's read
// loop exiting with a loss-class error.
func (m *mockSDR) failStream(err error) {
	m.mu.Lock()
	errCh := m.streamErrs
	ch := m.streamChan
	m.streaming = false
	m.mu.Unlock()
	if errCh == nil || ch == nil {
		return
	}
	errCh <- err
	close(ch)
	close(errCh)
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

// closeCount returns how many times the device was closed.
func (m *mockSDR) closeCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.closeCalls
}

// freqCallCount returns how many times the center frequency was set.
func (m *mockSDR) freqCallCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.freqCalls
}

// resetCallCount returns how many times the buffer was reset.
func (m *mockSDR) resetCallCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.resetCalls
}

var _ sdr.SDR = (*mockSDR)(nil)
