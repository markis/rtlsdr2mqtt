package decoder

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"testing"
	"time"

	"rtlsdr2mqtt/internal/config"
)

const (
	testMeterID     = "12345678"
	testProtocolSCM = "scm+"
)

func newTestLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
}

func newTestDecoder(t *testing.T, mock *mockSDR) *Decoder {
	t.Helper()
	cfg := &config.Config{
		SDR: config.SDRConfig{USBDevice: ""},
		Meters: []config.MeterConfig{
			{ID: testMeterID, Protocol: testProtocolSCM},
		},
	}
	d := NewDecoder(cfg, newTestLogger())
	d.sdr = mock
	return d
}

// ensureDecoderStarted starts the decoder, failing the test on error.
func ensureDecoderStarted(t *testing.T, d *Decoder) {
	t.Helper()
	if err := d.Start(context.Background()); err != nil {
		t.Fatalf("Start() failed: %v", err)
	}
}

func TestDecoderStartStreamingParams(t *testing.T) {
	mock := &mockSDR{}
	d := newTestDecoder(t, mock)
	ensureDecoderStarted(t, d)
	defer func() { _ = d.Stop() }()

	// Decode consumes exactly BlockSize2 bytes per call, delivered as samples (2 bytes each).
	wantBufLen := uint32(d.decoder.Cfg.BlockSize2 / 2)
	if mock.startBufLen != wantBufLen {
		t.Errorf("StartStreaming bufLen = %d, want %d", mock.startBufLen, wantBufLen)
	}
	if !mock.opened {
		t.Error("expected device to be opened")
	}
	if !mock.isStreaming() {
		t.Error("expected streaming to be active")
	}
}

func TestDecoderStopClosesStream(t *testing.T) {
	mock := &mockSDR{}
	d := newTestDecoder(t, mock)
	ensureDecoderStarted(t, d)

	if err := d.Stop(); err != nil {
		t.Fatalf("Stop() failed: %v", err)
	}

	if mock.isStreaming() {
		t.Error("expected streaming to be stopped")
	}
	if mock.isOpen() {
		t.Error("expected device to be closed")
	}
	if d.IsRunning() {
		t.Error("expected IsRunning() false after stop")
	}
}

func TestDecoderRestartAfterStop(t *testing.T) {
	mock := &mockSDR{}
	d := newTestDecoder(t, mock)
	ensureDecoderStarted(t, d)
	if err := d.Stop(); err != nil {
		t.Fatalf("Stop() failed: %v", err)
	}

	// Restart must produce a fresh, working pipeline.
	if err := d.Start(context.Background()); err != nil {
		t.Fatalf("restart Start() failed: %v", err)
	}
	defer func() { _ = d.Stop() }()

	if !d.IsRunning() {
		t.Error("expected IsRunning() true after restart")
	}
	if !mock.isStreaming() {
		t.Error("expected streaming active after restart")
	}
}

func TestWatchdogReportsStall(t *testing.T) {
	mock := &mockSDR{}
	d := newTestDecoder(t, mock)
	d.watchdogTimeout = 100 * time.Millisecond
	ensureDecoderStarted(t, d)
	defer func() { _ = d.Stop() }()

	msgChan, errChan, _ := d.Channels()

	// No blocks pushed: watchdog must fire and report the stall.
	select {
	case err := <-errChan:
		if !errors.Is(err, ErrSampleFlowStalled) {
			t.Errorf("expected ErrSampleFlowStalled, got %v", err)
		}
	case msg := <-msgChan:
		t.Fatalf("unexpected message during stall: %+v", msg)
	case <-time.After(5 * time.Second):
		t.Fatal("watchdog did not report the stall in time")
	}
}

func TestWatchdogSilentWhenSamplesFlow(t *testing.T) {
	mock := &mockSDR{}
	d := newTestDecoder(t, mock)
	d.watchdogTimeout = 250 * time.Millisecond
	ensureDecoderStarted(t, d)
	defer func() { _ = d.Stop() }()

	_, errChan, _ := d.Channels()

	block := make([]byte, d.decoder.Cfg.BlockSize2)
	timeout := time.After(2 * time.Second)
	ticker := time.NewTicker(50 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case err := <-errChan:
			t.Fatalf("watchdog fired while samples flowed: %v", err)
		case <-ticker.C:
			mock.push(block)
		case <-timeout:
			return // watchdog stayed silent for 2s of continuous samples
		}
	}
}

func TestDecoderHandlesStreamClose(t *testing.T) {
	mock := &mockSDR{}
	d := newTestDecoder(t, mock)
	ensureDecoderStarted(t, d)
	defer func() { _ = d.Stop() }()

	_, errChan, _ := d.Channels()

	// Close the stream out from under the decoder (simulates device death).
	mock.mu.Lock()
	ch := mock.streamChan
	mock.streaming = false
	mock.mu.Unlock()
	close(ch)

	// The decoder must report the closure so the controller restarts it.
	select {
	case err := <-errChan:
		if !errors.Is(err, ErrSampleStreamClosed) {
			t.Errorf("expected ErrSampleStreamClosed, got %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("decoder did not report stream closure in time")
	}
}
