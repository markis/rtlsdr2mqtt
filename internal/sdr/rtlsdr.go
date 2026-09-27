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
	"log/slog"
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
	dev         *C.rtlsdr_dev_t
	deviceIndex uint32
	mu          sync.Mutex
	isOpen      bool
	logger      *slog.Logger

	// Streaming state, guarded by mu.
	streaming       bool
	streamChan      chan []byte
	streamErrs      chan error
	closeStreamOnce sync.Once
	callbackID      uintptr
	asyncDone       chan struct{} // closed when rtlsdr_read_async returns
	cancelTimeout   time.Duration
}

// NewRTLSDRDevice creates a new RTL-SDR device instance with the specified device index.
func NewRTLSDRDevice(deviceIndex uint32) *RTLSDRDevice {
	return &RTLSDRDevice{
		deviceIndex:   deviceIndex,
		cancelTimeout: 5 * time.Second,
		logger:        slog.Default(),
	}
}

// SetLogger replaces the logger used for device-lifecycle messages.
func (d *RTLSDRDevice) SetLogger(logger *slog.Logger) {
	if logger == nil {
		return
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	d.logger = logger
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

	// If the async reader refuses to stop, the device handle must be leaked:
	// freeing it while the C read loop still runs is a use-after-free.
	if d.streaming {
		err := d.stopStreamLocked()
		if err != nil {
			return fmt.Errorf("cannot close device while streaming: %w", err)
		}
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

// SetGainMode sets the gainmode (manual or auto).
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

// StartStreaming begins async sample delivery. The returned stream's Samples
// channel receives IQ blocks and Errors receives at most one terminal stream
// error (for example a loss-class device failure) before Samples closes;
// Errors also closes so consumers can distinguish a failed stream from a
// clean stop. bufNum is the number of librtlsdr transfer buffers (0 uses the
// default: 15). bufLen is the size of each buffer in samples (0 uses the
// default: 16384). Sample blocks are dropped when the consumer cannot keep up.
func (d *RTLSDRDevice) StartStreaming(bufLen, bufNum uint32) (Stream, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	if !d.isOpen {
		return Stream{}, ErrDeviceNotOpen
	}
	if d.streaming {
		return Stream{}, ErrAlreadyStreaming
	}

	callback := func(samples []byte) {
		// Non-blocking delivery: the decoder tolerates dropped blocks far better
		// than an unbounded backlog or a blocked C read loop.
		select {
		case d.streamChan <- samples:
		default:
		}
	}

	callbackMu.Lock()
	d.callbackID = nextCallbackID
	nextCallbackID++
	callbackRefs[d.callbackID] = callback
	callbackMu.Unlock()

	// Small buffer: a few blocks of slack for the consumer, then drop. The
	// decoder tolerates dropped blocks far better than a blocked C read loop.
	d.streamChan = make(chan []byte, 4)
	d.streamErrs = make(chan error, 1)
	d.closeStreamOnce = sync.Once{}
	d.asyncDone = make(chan struct{})
	d.streaming = true

	go func() {
		// Note: This call blocks until rtlsdr_cancel_async is accepted, or
		// until the read loop dies on its own (lost device, bus error).
		ret := C.call_rtlsdr_read_async(
			d.dev,
			unsafe.Pointer(d.callbackID), //nolint:govet // Converting uintptr to unsafe.Pointer for CGO callback context
			C.uint32_t(bufNum),
			C.uint32_t(bufLen),
		)
		// Signal exit first: stopStreamLocked waits on this channel, and the
		// closed channel takes priority over further cancel attempts there.
		close(d.asyncDone)
		if ret == 0 {
			return
		}
		// The read loop died without cancellation: the device is lost or the
		// USB pipe failed. Deliver the classified error before closing the
		// sample channel so consumers leave recovery to the state machine
		// instead of stalling silently on a dead handle.
		d.mu.Lock()
		defer d.mu.Unlock()
		code := int(ret)
		if err := ClassifyLibusbError(code); err != nil {
			d.logger.Error("RTL-SDR sample stream lost device", "libusb_error", LibusbErrorName(code), "code", code)
			d.streamErrs <- err
		} else {
			d.logger.Error("RTL-SDR sample stream failed", "libusb_error", LibusbErrorName(code), "code", code)
			d.streamErrs <- fmt.Errorf("%w: libusb error %s (%d)", ErrAsyncReadFailed, LibusbErrorName(code), code)
		}
		d.closeStreamLocked()
	}()

	return Stream{Samples: d.streamChan, Errors: d.streamErrs}, nil
}

// closeStreamLocked closes the sample and error channels exactly once. The
// caller must hold d.mu.
func (d *RTLSDRDevice) closeStreamLocked() {
	d.closeStreamOnce.Do(func() {
		close(d.streamChan)
		close(d.streamErrs)
	})
}

// StopStreaming cancels the async sample reader and waits for it to exit.
// librtlsdr silently drops rtlsdr_cancel_async calls issued before the read
// loop reaches the RUNNING state, so cancellation is retried until the reader
// exits or the timeout elapses.
func (d *RTLSDRDevice) StopStreaming() error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if !d.streaming {
		return nil
	}

	err := d.stopStreamLocked()
	if err == nil {
		d.closeStreamLocked()
	}
	return err
}

// stopStreamLocked cancels and joins the async reader. The caller must hold d.mu.
// An already-exited read loop (asyncDone closed, for example after device loss)
// is detected first so cancellation is never attempted against a dead handle:
// the closed channel takes priority over further cancel attempts. On timeout
// the streaming flag stays set so Close refuses to free a device the C read
// loop still uses. It must not close streamChan in that case: stale references
// from the live callback would panic on send.
func (d *RTLSDRDevice) stopStreamLocked() error {
	callbackMu.Lock()
	delete(callbackRefs, d.callbackID)
	callbackMu.Unlock()
	d.callbackID = 0

	// Fast path: the read loop already exited (for example it died on device
	// loss before cancellation was ever issued).
	select {
	case <-d.asyncDone:
		d.streaming = false
		return nil
	default:
	}

	timeout := time.After(d.cancelTimeout)
	ticker := time.NewTicker(50 * time.Millisecond)
	defer ticker.Stop()

	var cancelErr error
	for {
		select {
		case <-d.asyncDone:
			d.streaming = false
			return nil
		case <-ticker.C:
			// Re-check before issuing cancellation: the read loop may have
			// exited since the last check; rtlsdr_cancel_async fails against
			// a dead handle, so it must not mask an exited loop.
			select {
			case <-d.asyncDone:
				d.streaming = false
				return nil
			default:
			}
			// Retry until the C read loop accepts the cancellation. A failing
			// cancel is kept as context, but the loop persists until
			// asyncDone or the timeout: this keeps the streaming flag (and
			// the Close guard) accurate on a wedged device.
			if ret := C.rtlsdr_cancel_async(d.dev); ret != 0 {
				cancelErr = fmt.Errorf("%w: error code %d", ErrCancelAsyncFailed, int(ret))
			}
		case <-timeout:
			// Final check: the read loop may have exited concurrently with
			// the timeout. Only then is a timeout reported.
			select {
			case <-d.asyncDone:
				d.streaming = false
				return nil
			default:
			}
			if cancelErr != nil {
				return fmt.Errorf("%w: %w", ErrStreamStopTimeout, cancelErr)
			}
			return ErrStreamStopTimeout
		}
	}
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
