// Package sdr provides interfaces and types for software-defined radio devices.
package sdr

import (
	"errors"
	"fmt"
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
	// ErrResetBufferFailed is returned when resetting the buffer fails.
	ErrResetBufferFailed = errors.New("failed to reset buffer")
	// ErrCancelAsyncFailed is returned when the async reader rejects cancellation.
	ErrCancelAsyncFailed = errors.New("failed to cancel async read")
	// ErrAlreadyStreaming is returned when starting a stream while one is active.
	ErrAlreadyStreaming = errors.New("device is already streaming")
	// ErrStreamStopTimeout is returned when the async sample reader does not
	// stop within the allotted time after cancellation.
	ErrStreamStopTimeout = errors.New("timed out stopping the sample stream")
	// ErrAsyncReadFailed is returned when the async sample reader exits with
	// a non-loss libusb error. Loss-class codes surface as ErrDeviceLost.
	ErrAsyncReadFailed = errors.New("async sample read failed")
	// ErrDeviceLost classifies loss-class libusb failures: the RTL-SDR vanished
	// from the USB bus (or the sample pipe died) and the handle is stale. The
	// receiver must stop using the handle and re-enumerate the device.
	ErrDeviceLost = errors.New("RTL-SDR device lost")
	// ErrDeviceAbsent is returned when fresh USB enumeration finds no matching
	// RTL-SDR device. It is retryable: the dongle may re-enumerate later.
	ErrDeviceAbsent = errors.New("no RTL-SDR device present on the USB bus")
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
	// StartStreaming begins async sample delivery. Samples arrive on the
	// returned stream's Samples channel; at most one terminal stream error
	// (for example a loss-class device failure) is delivered on its Errors
	// channel before Samples closes. Sample blocks are dropped when the
	// consumer cannot keep up.
	StartStreaming(bufLen, bufNum uint32) (Stream, error)
	// StopStreaming cancels sample delivery and closes the stream.
	// It returns an error if the async reader cannot be stopped.
	StopStreaming() error

	// Info
	GetDeviceCount() uint32
	GetDeviceName(index uint32) string
	GetTunerGains() []int
}

// Stream delivers async samples from an SDR device together with at most one
// terminal stream error. When the librtlsdr read loop dies on its own (device
// loss or bus error), the classified error is delivered on Errors before
// Samples closes, so consumers never stall silently on a dead handle.
type Stream struct {
	// Samples delivers IQ sample blocks; closed when the stream ends.
	Samples <-chan []byte
	// Errors receives at most one terminal stream error, then closes.
	Errors <-chan error
}

// libusb error codes that indicate device loss rather than a transient
// failure. librtlsdr propagates these codes from its async read loop.
const (
	libusbErrorIO       = -1 // LIBUSB_ERROR_IO
	libusbErrorNoDevice = -4 // LIBUSB_ERROR_NO_DEVICE
	libusbErrorNotFound = -5 // LIBUSB_ERROR_NOT_FOUND
)

// ClassifyLibusbError maps a negative libusb/rtlsdr return code to a
// loss-class error when the code indicates the device vanished from the bus.
// It returns nil for codes that do not indicate device loss.
func ClassifyLibusbError(code int) error {
	switch code {
	case libusbErrorIO, libusbErrorNoDevice, libusbErrorNotFound:
		return fmt.Errorf("%w (libusb error %d)", ErrDeviceLost, code)
	default:
		return nil
	}
}

// IsDeviceLost reports whether err is a loss-class device error: the RTL-SDR
// handle is stale and the device must be re-enumerated before reopening.
func IsDeviceLost(err error) bool {
	return errors.Is(err, ErrDeviceLost)
}

// LibusbErrorName returns the symbolic name of a libusb error code for logs.
// Unknown codes report as "LIBUSB_ERROR_UNKNOWN".
func LibusbErrorName(code int) string {
	switch code {
	case libusbErrorIO:
		return "LIBUSB_ERROR_IO"
	case -2:
		return "LIBUSB_ERROR_INVALID_PARAM"
	case -3:
		return "LIBUSB_ERROR_ACCESS"
	case libusbErrorNoDevice:
		return "LIBUSB_ERROR_NO_DEVICE"
	case libusbErrorNotFound:
		return "LIBUSB_ERROR_NOT_FOUND"
	case -6:
		return "LIBUSB_ERROR_BUSY"
	case -7:
		return "LIBUSB_ERROR_TIMEOUT"
	case -8:
		return "LIBUSB_ERROR_OVERFLOW"
	case -9:
		return "LIBUSB_ERROR_PIPE"
	case -11:
		return "LIBUSB_ERROR_INTERRUPTED"
	case -12:
		return "LIBUSB_ERROR_NO_MEM"
	case -99:
		return "LIBUSB_ERROR_OTHER"
	default:
		return "LIBUSB_ERROR_UNKNOWN"
	}
}

// DeviceSetup carries the pipeline-owned SDR parameters applied on every
// device acquisition, so startup and re-enumeration recovery configure the
// device identically.
type DeviceSetup struct {
	CenterFreq uint32
	SampleRate uint32
}

// DeviceSource acquires opened RTL-SDR devices through fresh USB enumeration.
// The startup path and the re-enumeration recovery path share implementations
// of this interface; tests substitute fakes to exercise recovery without
// hardware.
type DeviceSource interface {
	// Acquire returns an opened device and a short identity string for logs
	// (for example "Generic RTL2832U (index 0)"). It returns ErrDeviceAbsent
	// when no matching device is currently enumerated; acquisition errors
	// after a device was found (busy, setup failure) are returned as-is.
	Acquire() (SDR, string, error)
}

// SampleCallback is called for each buffer of IQ samples in async mode.
type SampleCallback func(samples []byte)
