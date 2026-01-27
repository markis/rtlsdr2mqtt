//go:build !cgo || !linux

package sdr

import (
	"errors"
	"testing"
	"time"
)

func TestRTLSDRDevice_NoCGO_Basic(t *testing.T) {
	device := NewRTLSDRDevice(0)
	if device == nil {
		t.Fatal("NewRTLSDRDevice() returned nil")
	}

	// Test GetDeviceCount
	if count := device.GetDeviceCount(); count != 0 {
		t.Errorf("GetDeviceCount() = %v, want 0", count)
	}

	// Test GetDeviceName
	if name := device.GetDeviceName(0); name != "" {
		t.Errorf("GetDeviceName() = %q, want empty string", name)
	}

	// Test Open
	if err := device.Open(); !errors.Is(err, ErrNoCGO) {
		t.Errorf("Open() error = %v, want %v", err, ErrNoCGO)
	}

	// Test Close
	if err := device.Close(); err != nil {
		t.Errorf("Close() error = %v, want nil", err)
	}
}

func TestRTLSDRDevice_NoCGO_Configuration(t *testing.T) {
	device := NewRTLSDRDevice(0)

	// Test SetCenterFreq
	if err := device.SetCenterFreq(915000000); !errors.Is(err, ErrNoCGO) {
		t.Errorf("SetCenterFreq() error = %v, want %v", err, ErrNoCGO)
	}

	// Test SetSampleRate
	if err := device.SetSampleRate(2048000); !errors.Is(err, ErrNoCGO) {
		t.Errorf("SetSampleRate() error = %v, want %v", err, ErrNoCGO)
	}

	// Test SetGainMode
	if err := device.SetGainMode(true); !errors.Is(err, ErrNoCGO) {
		t.Errorf("SetGainMode() error = %v, want %v", err, ErrNoCGO)
	}

	// Test SetGain
	if err := device.SetGain(496); !errors.Is(err, ErrNoCGO) {
		t.Errorf("SetGain() error = %v, want %v", err, ErrNoCGO)
	}

	// Test SetAGCMode
	if err := device.SetAGCMode(true); !errors.Is(err, ErrNoCGO) {
		t.Errorf("SetAGCMode() error = %v, want %v", err, ErrNoCGO)
	}

	// Test SetFreqCorrection
	if err := device.SetFreqCorrection(10); !errors.Is(err, ErrNoCGO) {
		t.Errorf("SetFreqCorrection() error = %v, want %v", err, ErrNoCGO)
	}
}

func TestRTLSDRDevice_NoCGO_IO(t *testing.T) {
	device := NewRTLSDRDevice(0)

	// Test ResetBuffer
	if err := device.ResetBuffer(); !errors.Is(err, ErrNoCGO) {
		t.Errorf("ResetBuffer() error = %v, want %v", err, ErrNoCGO)
	}

	// Test ReadSync
	buf := make([]byte, 1024)
	if n, err := device.ReadSync(buf); !errors.Is(err, ErrNoCGO) {
		t.Errorf("ReadSync() error = %v, want %v", err, ErrNoCGO)
	} else if n != 0 {
		t.Errorf("ReadSync() n = %v, want 0", n)
	}

	// Test StartAsync
	callback := func(_ []byte) {}
	if err := device.StartAsync(callback, 0, 0); !errors.Is(err, ErrNoCGO) {
		t.Errorf("StartAsync() error = %v, want %v", err, ErrNoCGO)
	}

	// Test CancelAsync
	if err := device.CancelAsync(); !errors.Is(err, ErrNoCGO) {
		t.Errorf("CancelAsync() error = %v, want %v", err, ErrNoCGO)
	}

	// Test GetTunerGains
	if gains := device.GetTunerGains(); gains != nil {
		t.Errorf("GetTunerGains() = %v, want nil", gains)
	}

	// Test SetDeadline
	if err := device.SetDeadline(time.Now()); err != nil {
		t.Errorf("SetDeadline() error = %v, want nil", err)
	}
}
