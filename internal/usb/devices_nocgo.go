//go:build !cgo

// Package usb provides USB device management functions.
package usb

import (
	"fmt"
	"log/slog"
)

// SupportedDeviceIDs contains the list of supported RTL-SDR device IDs
var SupportedDeviceIDs = []string{
	"0458:707f",
	"048d:9135",
	"0bda:2832",
	"0bda:2838",
	"0ccd:00a9",
	"0ccd:00b3",
	"0ccd:00d3",
	"0ccd:00e0",
	"185b:0620",
	"185b:0650",
	"1b80:d393",
	"1b80:d394",
	"1b80:d395",
	"1b80:d39d",
	"1b80:d3a4",
	"1d19:1101",
	"1d19:1102",
	"1d19:1103",
	"1f4d:b803",
	"1f4d:c803",
	"1f4d:d803",
}

// DeviceInfo represents information about a USB device (stub for no-CGO builds)
type DeviceInfo struct {
	BusNumber     int
	DeviceAddress int
	VendorID      uint16
	ProductID     uint16
	BusDevice     string // Format: "BUS:DEV" (e.g., "001:003")
}

// String returns a string representation of the device
func (d DeviceInfo) String() string {
	return fmt.Sprintf("%s (%04x:%04x)", d.BusDevice, d.VendorID, d.ProductID)
}

// FindRTLSDRDevices returns a mock device list when CGO is disabled
func FindRTLSDRDevices(logger *slog.Logger) ([]DeviceInfo, error) {
	if logger == nil {
		logger = slog.Default()
	}

	logger.Warn("USB device detection not available (built without CGO)")

	// Return a mock device for testing/demo purposes
	mockDevice := DeviceInfo{
		BusNumber:     1,
		DeviceAddress: 3,
		VendorID:      0x0bda,
		ProductID:     0x2838,
		BusDevice:     "001:003",
	}

	return []DeviceInfo{mockDevice}, nil
}

// GetFirstDevice returns a mock device when CGO is disabled
func GetFirstDevice(logger *slog.Logger) (*DeviceInfo, error) {
	devices, err := FindRTLSDRDevices(logger)
	if err != nil {
		return nil, err
	}

	if len(devices) == 0 {
		return nil, fmt.Errorf("no RTL-SDR devices found (CGO disabled)")
	}

	return &devices[0], nil
}

// FindDeviceByBusDevice returns a mock device when CGO is disabled
func FindDeviceByBusDevice(busDevice string, logger *slog.Logger) (*DeviceInfo, error) {
	if logger == nil {
		logger = slog.Default()
	}

	logger.Warn("USB device search not available (built without CGO)", "requested", busDevice)

	// Return a mock device that matches the requested bus:device
	mockDevice := DeviceInfo{
		BusNumber:     1,
		DeviceAddress: 3,
		VendorID:      0x0bda,
		ProductID:     0x2838,
		BusDevice:     busDevice,
	}

	return &mockDevice, nil
}
