//go:build !cgo || !linux

package usb

import (
	"log/slog"
)

// ResetUSBDevice is a no-op when CGO is disabled or not on Linux.
func ResetUSBDevice(busDevice string, logger *slog.Logger) error {
	if logger == nil {
		logger = slog.Default()
	}

	logger.Warn("USB device reset not available (built without CGO or not on Linux)", "device", busDevice)
	return nil // Non-fatal for compatibility
}

// ResetFirstDevice is a no-op when CGO is disabled or not on Linux.
func ResetFirstDevice(logger *slog.Logger) error {
	if logger == nil {
		logger = slog.Default()
	}

	logger.Warn("USB device reset not available (built without CGO or not on Linux)")
	return nil // Non-fatal for compatibility
}

// ResetDeviceByID is a no-op when CGO is disabled or not on Linux.
func ResetDeviceByID(deviceID string, logger *slog.Logger) error {
	if logger == nil {
		logger = slog.Default()
	}

	logger.Warn("USB device reset not available (built without CGO or not on Linux)", "device_id", deviceID)
	return nil // Non-fatal for compatibility
}
