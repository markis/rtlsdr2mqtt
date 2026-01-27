package sdr

import (
	"errors"
	"testing"
)

func TestErrorDefinitions(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want string
	}{
		{
			name: "ErrDeviceNotOpen",
			err:  ErrDeviceNotOpen,
			want: "device not open",
		},
		{
			name: "ErrOpenFailed",
			err:  ErrOpenFailed,
			want: "failed to open device",
		},
		{
			name: "ErrSetFreqFailed",
			err:  ErrSetFreqFailed,
			want: "failed to set center frequency",
		},
		{
			name: "ErrSetRateFailed",
			err:  ErrSetRateFailed,
			want: "failed to set sample rate",
		},
		{
			name: "ErrSetGainModeFailed",
			err:  ErrSetGainModeFailed,
			want: "failed to set gain mode",
		},
		{
			name: "ErrSetGainFailed",
			err:  ErrSetGainFailed,
			want: "failed to set gain",
		},
		{
			name: "ErrSetAGCFailed",
			err:  ErrSetAGCFailed,
			want: "failed to set AGC mode",
		},
		{
			name: "ErrSetFreqCorrectionFailed",
			err:  ErrSetFreqCorrectionFailed,
			want: "failed to set frequency correction",
		},
		{
			name: "ErrReadFailed",
			err:  ErrReadFailed,
			want: "read failed",
		},
		{
			name: "ErrAsyncFailed",
			err:  ErrAsyncFailed,
			want: "async operation failed",
		},
		{
			name: "ErrResetBufferFailed",
			err:  ErrResetBufferFailed,
			want: "failed to reset buffer",
		},
		{
			name: "ErrCancelAsyncFailed",
			err:  ErrCancelAsyncFailed,
			want: "failed to cancel async",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.err == nil {
				t.Error("error is nil")
				return
			}
			if got := tt.err.Error(); got != tt.want {
				t.Errorf("error.Error() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestErrorWrapping(t *testing.T) {
	// Test that errors can be wrapped and unwrapped
	additionalErr := ErrReadFailed
	wrappedErr := errors.Join(ErrDeviceNotOpen, additionalErr)

	if !errors.Is(wrappedErr, ErrDeviceNotOpen) {
		t.Error("errors.Is() failed to detect wrapped error")
	}
}

func TestSampleCallback(t *testing.T) {
	// Test that SampleCallback type works
	called := false
	var callback SampleCallback = func(samples []byte) {
		called = true
		if len(samples) != 10 {
			t.Errorf("callback received %d bytes, want 10", len(samples))
		}
	}

	// Invoke the callback
	testData := make([]byte, 10)
	callback(testData)

	if !called {
		t.Error("callback was not invoked")
	}
}
