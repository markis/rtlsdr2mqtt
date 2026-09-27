package sdr

import (
	"errors"
	"fmt"
	"testing"
)

// errTestBoom is a synthetic generic error for classification tests.
var errTestBoom = errors.New("test boom")

func TestClassifyLibusbError(t *testing.T) {
	tests := []struct {
		name       string
		code       int
		wantLoss   bool
		wantSymbol string
	}{
		{"IO is loss", -1, true, "LIBUSB_ERROR_IO"},
		{"NoDevice is loss", -4, true, "LIBUSB_ERROR_NO_DEVICE"},
		{"NotFound is loss", -5, true, "LIBUSB_ERROR_NOT_FOUND"},
		{"InvalidParam is not loss", -2, false, "LIBUSB_ERROR_INVALID_PARAM"},
		{"Access is not loss", -3, false, "LIBUSB_ERROR_ACCESS"},
		{"Busy is not loss", -6, false, "LIBUSB_ERROR_BUSY"},
		{"Timeout is not loss", -7, false, "LIBUSB_ERROR_TIMEOUT"},
		{"zero is not loss", 0, false, "LIBUSB_ERROR_UNKNOWN"},
		{"unknown code is not loss", -42, false, "LIBUSB_ERROR_UNKNOWN"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ClassifyLibusbError(tt.code)
			if tt.wantLoss {
				if err == nil {
					t.Fatalf("ClassifyLibusbError(%d) = nil, want loss-class error", tt.code)
				}
				if !errors.Is(err, ErrDeviceLost) {
					t.Errorf("ClassifyLibusbError(%d) = %v, want error wrapping ErrDeviceLost", tt.code, err)
				}
			} else if err != nil {
				t.Errorf("ClassifyLibusbError(%d) = %v, want nil", tt.code, err)
			}

			if got := LibusbErrorName(tt.code); got != tt.wantSymbol {
				t.Errorf("LibusbErrorName(%d) = %q, want %q", tt.code, got, tt.wantSymbol)
			}
		})
	}
}

func TestIsDeviceLost(t *testing.T) {
	lossErr := fmt.Errorf("async sample read failed: %w", ErrDeviceLost)
	doubleWrapped := fmt.Errorf("sample stream ended: %w", lossErr)

	tests := []struct {
		name string
		err  error
		want bool
	}{
		{"nil is not loss", nil, false},
		{"plain loss sentinel", ErrDeviceLost, true},
		{"wrapped loss is loss", lossErr, true},
		{"double-wrapped loss is loss", doubleWrapped, true},
		{"open failure is not loss", ErrOpenFailed, false},
		{"stream stop timeout is not loss", ErrStreamStopTimeout, false},
		{"device not open is not loss", ErrDeviceNotOpen, false},
		{"generic error is not loss", errTestBoom, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsDeviceLost(tt.err); got != tt.want {
				t.Errorf("IsDeviceLost(%v) = %v, want %v", tt.err, got, tt.want)
			}
		})
	}
}
