//go:build !cgo || !linux

// Package sdr provides a stub implementation when CGO is not available.
package sdr

import (
	"errors"
	"time"
)

// ErrNoCGO is returned when trying to use direct RTL-SDR access without CGO.
var ErrNoCGO = errors.New("direct RTL-SDR access requires CGO (build with CGO_ENABLED=1)")

// RTLSDRDevice is a stub implementation for non-CGO builds.
type RTLSDRDevice struct {
	deviceIndex uint32
}

// NewRTLSDRDevice creates a stub RTL-SDR device that always returns ErrNoCGO.
func NewRTLSDRDevice(deviceIndex uint32) *RTLSDRDevice {
	return &RTLSDRDevice{
		deviceIndex: deviceIndex,
	}
}

// GetDeviceCount always returns 0 without CGO.
func (d *RTLSDRDevice) GetDeviceCount() uint32 {
	return 0
}

// GetDeviceName always returns empty string without CGO.
func (d *RTLSDRDevice) GetDeviceName(_ uint32) string {
	return ""
}

// Open returns ErrNoCGO.
func (d *RTLSDRDevice) Open() error {
	return ErrNoCGO
}

// Close is a no-op without CGO.
func (d *RTLSDRDevice) Close() error {
	return nil
}

// SetCenterFreq returns ErrNoCGO.
func (d *RTLSDRDevice) SetCenterFreq(_ uint32) error {
	return ErrNoCGO
}

// SetSampleRate returns ErrNoCGO.
func (d *RTLSDRDevice) SetSampleRate(_ uint32) error {
	return ErrNoCGO
}

// SetGainMode returns ErrNoCGO.
func (d *RTLSDRDevice) SetGainMode(_ bool) error {
	return ErrNoCGO
}

// SetGain returns ErrNoCGO.
func (d *RTLSDRDevice) SetGain(_ int) error {
	return ErrNoCGO
}

// SetAGCMode returns ErrNoCGO.
func (d *RTLSDRDevice) SetAGCMode(_ bool) error {
	return ErrNoCGO
}

// SetFreqCorrection returns ErrNoCGO.
func (d *RTLSDRDevice) SetFreqCorrection(_ int) error {
	return ErrNoCGO
}

// ResetBuffer returns ErrNoCGO.
func (d *RTLSDRDevice) ResetBuffer() error {
	return ErrNoCGO
}

// ReadSync returns ErrNoCGO.
func (d *RTLSDRDevice) ReadSync(_ []byte) (int, error) {
	return 0, ErrNoCGO
}

// StartAsync returns ErrNoCGO.
func (d *RTLSDRDevice) StartAsync(_ SampleCallback, _, _ uint32) error {
	return ErrNoCGO
}

// CancelAsync returns ErrNoCGO.
func (d *RTLSDRDevice) CancelAsync() error {
	return ErrNoCGO
}

// GetTunerGains returns nil without CGO.
func (d *RTLSDRDevice) GetTunerGains() []int {
	return nil
}

// SetDeadline is a no-op without CGO.
func (d *RTLSDRDevice) SetDeadline(_ time.Time) error {
	return nil
}
