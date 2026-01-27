//go:build cgo && linux

// Package sdr provides direct librtlsdr integration via CGO.
package sdr

/*
#cgo LDFLAGS: -lrtlsdr
#include <rtl-sdr.h>
#include <stdlib.h>
#include <stdint.h>

// Callback wrapper for async reads
extern void goSampleCallback(unsigned char *buf, uint32_t len, void *ctx);

// Helper to call rtlsdr_read_async with our callback
static inline int call_rtlsdr_read_async(rtlsdr_dev_t *dev, void *ctx, uint32_t buf_num, uint32_t buf_len) {
    return rtlsdr_read_async(dev, goSampleCallback, ctx, buf_num, buf_len);
}
*/
import "C" //nolint:gocritic // CGO import is separate from standard imports

import (
	"fmt"
	"sync"
	"time"
	"unsafe" //nolint:gocritic // Required for CGO, not a duplicate import
)

var (
	// Global map to track callback contexts.
	callbackMu     sync.Mutex
	callbackRefs           = make(map[uintptr]SampleCallback)
	nextCallbackID uintptr = 1
)

// RTLSDRDevice implements the SDR interface using direct librtlsdr C bindings.
type RTLSDRDevice struct {
	dev          *C.rtlsdr_dev_t
	callback     SampleCallback
	callbackID   uintptr
	mu           sync.Mutex
	isOpen       bool
	deviceIndex  uint32
	readDeadline time.Time
}

// NewRTLSDRDevice creates a new RTL-SDR device instance with the specified device index.
func NewRTLSDRDevice(deviceIndex uint32) *RTLSDRDevice {
	return &RTLSDRDevice{
		deviceIndex: deviceIndex,
	}
}

// GetDeviceCount returns the number of RTL-SDR devices found.
func (d *RTLSDRDevice) GetDeviceCount() uint32 {
	return uint32(C.rtlsdr_get_device_count())
}

// GetDeviceName returns the name of the device at the given index.
func (d *RTLSDRDevice) GetDeviceName(index uint32) string {
	cName := C.rtlsdr_get_device_name(C.uint32_t(index))
	if cName == nil {
		return ""
	}
	return C.GoString(cName)
}

// Open opens the RTL-SDR device using the device index set during initialization.
func (d *RTLSDRDevice) Open() error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.isOpen {
		return nil
	}

	ret := C.rtlsdr_open(&d.dev, C.uint32_t(d.deviceIndex)) //nolint:gocritic // False positive: CGO function call
	if ret != 0 {
		return fmt.Errorf("%w: error code %d", ErrOpenFailed, ret)
	}

	d.isOpen = true
	return nil
}

// Close closes the RTL-SDR device.
func (d *RTLSDRDevice) Close() error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if !d.isOpen {
		return nil
	}

	// Clean up callback registration if any
	if d.callbackID != 0 {
		callbackMu.Lock()
		delete(callbackRefs, d.callbackID)
		callbackMu.Unlock()
		d.callbackID = 0
	}

	C.rtlsdr_close(d.dev)
	d.isOpen = false
	d.dev = nil
	return nil
}

// SetCenterFreq sets the center frequency in Hz.
func (d *RTLSDRDevice) SetCenterFreq(freq uint32) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if !d.isOpen {
		return ErrDeviceNotOpen
	}

	ret := C.rtlsdr_set_center_freq(d.dev, C.uint32_t(freq))
	if ret != 0 {
		return fmt.Errorf("%w: error code %d", ErrSetFreqFailed, ret)
	}
	return nil
}

// SetSampleRate sets the sample rate in Hz.
func (d *RTLSDRDevice) SetSampleRate(rate uint32) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if !d.isOpen {
		return ErrDeviceNotOpen
	}

	ret := C.rtlsdr_set_sample_rate(d.dev, C.uint32_t(rate))
	if ret != 0 {
		return fmt.Errorf("%w: error code %d", ErrSetRateFailed, ret)
	}
	return nil
}

// SetGainMode sets the gain mode (manual or auto).
// If manual is true, manual gain mode is enabled. Otherwise, auto gain is used.
func (d *RTLSDRDevice) SetGainMode(manual bool) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if !d.isOpen {
		return ErrDeviceNotOpen
	}

	mode := C.int(0)
	if manual {
		mode = C.int(1)
	}

	ret := C.rtlsdr_set_tuner_gain_mode(d.dev, mode)
	if ret != 0 {
		return fmt.Errorf("%w: error code %d", ErrSetGainModeFailed, ret)
	}
	return nil
}

// SetGain sets the tuner gain in tenths of dB (e.g., 496 means 49.6 dB).
func (d *RTLSDRDevice) SetGain(gain int) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if !d.isOpen {
		return ErrDeviceNotOpen
	}

	ret := C.rtlsdr_set_tuner_gain(d.dev, C.int(gain))
	if ret != 0 {
		return fmt.Errorf("%w: error code %d", ErrSetGainFailed, ret)
	}
	return nil
}

// SetAGCMode enables or disables the RTL2832's AGC mode.
func (d *RTLSDRDevice) SetAGCMode(enabled bool) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if !d.isOpen {
		return ErrDeviceNotOpen
	}

	mode := C.int(0)
	if enabled {
		mode = C.int(1)
	}

	ret := C.rtlsdr_set_agc_mode(d.dev, mode)
	if ret != 0 {
		return fmt.Errorf("%w: error code %d", ErrSetAGCFailed, ret)
	}
	return nil
}

// SetFreqCorrection sets the frequency correction in parts per million (PPM).
func (d *RTLSDRDevice) SetFreqCorrection(ppm int) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if !d.isOpen {
		return ErrDeviceNotOpen
	}

	ret := C.rtlsdr_set_freq_correction(d.dev, C.int(ppm))
	if ret != 0 {
		return fmt.Errorf("%w: error code %d", ErrSetFreqCorrectionFailed, ret)
	}
	return nil
}

// ResetBuffer resets the sample buffer.
func (d *RTLSDRDevice) ResetBuffer() error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if !d.isOpen {
		return ErrDeviceNotOpen
	}

	ret := C.rtlsdr_reset_buffer(d.dev)
	if ret != 0 {
		return fmt.Errorf("%w: error code %d", ErrResetBufferFailed, ret)
	}
	return nil
}

// ReadSync performs a synchronous read of IQ samples into the provided buffer.
// Returns the number of bytes actually read.
func (d *RTLSDRDevice) ReadSync(buf []byte) (int, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	if !d.isOpen {
		return 0, ErrDeviceNotOpen
	}

	if len(buf) == 0 {
		return 0, nil
	}

	var nRead C.int
	ret := C.rtlsdr_read_sync(
		d.dev,
		unsafe.Pointer(&buf[0]),
		C.int(len(buf)),
		&nRead,
	)

	if ret != 0 {
		return 0, fmt.Errorf("%w: error code %d", ErrReadFailed, ret)
	}

	return int(nRead), nil
}

// StartAsync starts asynchronous sample reading with a callback.
// bufNum is the number of buffers to allocate (use 0 for default: 15).
// bufLen is the size of each buffer (use 0 for default: 16384).
func (d *RTLSDRDevice) StartAsync(callback SampleCallback, bufNum, bufLen uint32) error {
	d.mu.Lock()

	if !d.isOpen {
		d.mu.Unlock()
		return ErrDeviceNotOpen
	}

	// Register callback
	callbackMu.Lock()
	d.callbackID = nextCallbackID
	nextCallbackID++
	callbackRefs[d.callbackID] = callback
	callbackMu.Unlock()

	d.callback = callback
	d.mu.Unlock()

	// Note: This call blocks until CancelAsync is called
	ret := C.call_rtlsdr_read_async(
		d.dev,
		unsafe.Pointer(d.callbackID), //nolint:govet // Converting uintptr to unsafe.Pointer for CGO callback context
		C.uint32_t(bufNum),
		C.uint32_t(bufLen),
	)

	if ret != 0 {
		return fmt.Errorf("%w: error code %d", ErrAsyncFailed, ret)
	}

	return nil
}

// CancelAsync cancels the asynchronous reading operation.
func (d *RTLSDRDevice) CancelAsync() error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if !d.isOpen {
		return ErrDeviceNotOpen
	}

	ret := C.rtlsdr_cancel_async(d.dev)
	if ret != 0 {
		return fmt.Errorf("%w: error code %d", ErrCancelAsyncFailed, ret)
	}

	return nil
}

// GetTunerGains returns the list of available tuner gains in tenths of dB.
func (d *RTLSDRDevice) GetTunerGains() []int {
	d.mu.Lock()
	defer d.mu.Unlock()

	if !d.isOpen {
		return nil
	}

	// First call to get the count
	count := C.rtlsdr_get_tuner_gains(d.dev, nil)
	if count <= 0 {
		return nil
	}

	// Allocate buffer and get gains
	gains := make([]C.int, count)
	C.rtlsdr_get_tuner_gains(d.dev, &gains[0])

	// Convert to Go slice
	result := make([]int, count)
	for i := range int(count) {
		result[i] = int(gains[i])
	}

	return result
}

// SetDeadline sets a read deadline for compatibility with io.Reader expectations.
// Note: librtlsdr doesn't support native deadlines, so this is a no-op for now.
// The decoder loop handles timeouts at a higher level.
func (d *RTLSDRDevice) SetDeadline(t time.Time) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	d.readDeadline = t
	return nil
}

// goSampleCallback is called by C code for async sample delivery.
//
//export goSampleCallback
func goSampleCallback(buf *C.uchar, length C.uint32_t, ctx unsafe.Pointer) {
	callbackID := uintptr(ctx)

	callbackMu.Lock()
	callback, ok := callbackRefs[callbackID]
	callbackMu.Unlock()

	if !ok {
		return
	}

	// Convert C buffer to Go slice
	samples := C.GoBytes(unsafe.Pointer(buf), C.int(length))
	callback(samples)
}
