package sdr

import (
	"errors"
	"log/slog"
	"os"
	"testing"

	"rtlsdr2mqtt/internal/config"
)

func testAcquirerLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
}

func TestNewDeviceSource(t *testing.T) {
	cfg := &config.Config{}
	src := NewDeviceSource(cfg, testAcquirerLogger())
	if src == nil {
		t.Fatal("NewDeviceSource() returned nil")
	}
	if src.cfg != cfg {
		t.Error("acquirer config not set correctly")
	}
	if src.newDevice == nil {
		t.Error("acquirer device factory not set")
	}
}

func TestDeviceSourceAcquireNoDevice(t *testing.T) {
	// Without CGO there are no enumerated devices, so acquisition must fail
	// with the retryable absent error rather than a hard failure.
	cfg := &config.Config{}
	src := NewDeviceSource(cfg, testAcquirerLogger())

	device, identity, err := src.Acquire()
	if !errors.Is(err, ErrDeviceAbsent) {
		t.Fatalf("Acquire() error = %v, want %v", err, ErrDeviceAbsent)
	}
	if device != nil {
		t.Errorf("Acquire() device = %v, want nil", device)
	}
	if identity != "" {
		t.Errorf("Acquire() identity = %q, want empty", identity)
	}
}
