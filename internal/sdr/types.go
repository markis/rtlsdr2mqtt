// Package sdr provides interfaces and types for software-defined radio devices.
package sdr

import (
	"errors"
	"time"
)

var (
	// ErrDeviceNotOpen is returned when attempting operations on a closed device.
	ErrDeviceNotOpen = errors.New("device not open")
	// ErrOpenFailed is returned when device open fails.
	ErrOpenFailed = errors.New("failed to open device")
	// ErrSetFreqFailed is returned when setting center frequency fails.
	ErrSetFreqFailed = errors.New("failed to set center frequency")
	// ErrSetRateFailed is returned when setting sample rate fails.
	ErrSetRateFailed = errors.New("failed to set sample rate")
	// ErrSetGainModeFailed is returned when setting gain mode fails.
	ErrSetGainModeFailed = errors.New("failed to set gain mode")
	// ErrSetGainFailed is returned when setting gain fails.
	ErrSetGainFailed = errors.New("failed to set gain")
	// ErrSetAGCFailed is returned when setting AGC mode fails.
	ErrSetAGCFailed = errors.New("failed to set AGC mode")
	// ErrSetFreqCorrectionFailed is returned when setting frequency correction fails.
	ErrSetFreqCorrectionFailed = errors.New("failed to set frequency correction")
	// ErrReadFailed is returned when reading samples fails.
	ErrReadFailed = errors.New("read failed")
	// ErrAsyncFailed is returned when async operation fails.
	ErrAsyncFailed = errors.New("async operation failed")
	// ErrResetBufferFailed is returned when resetting the buffer fails.
	ErrResetBufferFailed = errors.New("failed to reset buffer")
	// ErrCancelAsyncFailed is returned when canceling async operation fails.
	ErrCancelAsyncFailed = errors.New("failed to cancel async")
)

// SDR defines the interface for software-defined radio devices.
//
//nolint:interfacebloat // SDR interface needs many methods to fully support librtlsdr API
type SDR interface {
	// Device management
	Open() error
	Close() error

	// Configuration
	SetCenterFreq(freq uint32) error
	SetSampleRate(rate uint32) error
	SetGainMode(manual bool) error
	SetGain(gain int) error
	SetAGCMode(enabled bool) error
	SetFreqCorrection(ppm int) error

	// Streaming
	ResetBuffer() error
	ReadSync(buf []byte) (int, error)
	StartAsync(callback SampleCallback, bufNum, bufLen uint32) error
	CancelAsync() error

	// Info
	GetDeviceCount() uint32
	GetDeviceName(index uint32) string
	GetTunerGains() []int

	// Legacy compatibility for deadline setting (like rtltcp.SDR)
	SetDeadline(t time.Time) error
}

// SampleCallback is called for each buffer of IQ samples in async mode.
type SampleCallback func(samples []byte)
