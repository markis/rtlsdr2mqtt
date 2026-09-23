// Package sdr provides interfaces and types for software-defined radio devices.
package sdr

import "errors"

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
	// ErrResetBufferFailed is returned when resetting the buffer fails.
	ErrResetBufferFailed = errors.New("failed to reset buffer")
	// ErrCancelAsyncFailed is returned when the async reader rejects cancellation.
	ErrCancelAsyncFailed = errors.New("failed to cancel async read")
	// ErrAlreadyStreaming is returned when starting a stream while one is active.
	ErrAlreadyStreaming = errors.New("device is already streaming")
	// ErrStreamStopTimeout is returned when the async sample reader does not
	// stop within the allotted time after cancellation.
	ErrStreamStopTimeout = errors.New("timed out stopping the sample stream")
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
	// StartStreaming begins async sample delivery to the returned channel.
	// Sample blocks are dropped when the consumer cannot keep up.
	StartStreaming(bufLen, bufNum uint32) (<-chan []byte, error)
	// StopStreaming cancels sample delivery and closes the stream.
	// It returns an error if the async reader cannot be stopped.
	StopStreaming() error

	// Info
	GetDeviceCount() uint32
	GetDeviceName(index uint32) string
	GetTunerGains() []int
}

// SampleCallback is called for each buffer of IQ samples in async mode.
type SampleCallback func(samples []byte)
