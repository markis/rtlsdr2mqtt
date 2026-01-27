//go:build cgo

// Package usb provides USB device management functions.
package usb

import (
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"github.com/google/gousb"
)

var (
	ErrNoRTLSDRDevicesFound = errors.New("no RTL-SDR devices found")
	ErrDeviceNotFound       = errors.New("device not found")
)

// SupportedDeviceIDs contains the list of supported RTL-SDR device IDs.
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

// DeviceInfo represents information about a USB device.
type DeviceInfo struct {
	BusNumber     int
	DeviceAddress int
	VendorID      uint16
	ProductID     uint16
	BusDevice     string // Format: "BUS:DEV" (e.g., "001:003")
}

// String returns a string representation of the device.
func (d DeviceInfo) String() string {
	return fmt.Sprintf("%s (%04x:%04x)", d.BusDevice, d.VendorID, d.ProductID)
}

// FindRTLSDRDevices returns a list of connected RTL-SDR devices.
func FindRTLSDRDevices(logger *slog.Logger) ([]DeviceInfo, error) {
	if logger == nil {
		logger = slog.Default()
	}

	ctx := gousb.NewContext()
	defer ctx.Close()

	devices := make([]DeviceInfo, 0, 8) // Pre-allocate for typical number of devices

	// Open devices that match supported device IDs
	devs, err := ctx.OpenDevices(func(desc *gousb.DeviceDesc) bool {
		deviceID := fmt.Sprintf("%04x:%04x", desc.Vendor, desc.Product)
		for _, supported := range SupportedDeviceIDs {
			if strings.EqualFold(deviceID, supported) {
				return true
			}
		}
		return false
	})
	if err != nil {
		return nil, fmt.Errorf("failed to open USB devices: %w", err)
	}

	// Process found devices
	for _, dev := range devs {
		desc := dev.Desc

		deviceInfo := DeviceInfo{
			BusNumber:     desc.Bus,
			DeviceAddress: desc.Address,
			VendorID:      uint16(desc.Vendor),
			ProductID:     uint16(desc.Product),
			BusDevice:     fmt.Sprintf("%03d:%03d", desc.Bus, desc.Address),
		}

		devices = append(devices, deviceInfo)
		logger.Debug("Found RTL-SDR device", "device", deviceInfo.String())

		dev.Close()
	}

	logger.Info("RTL-SDR device scan complete", "found", len(devices))
	return devices, nil
}

// GetFirstDevice returns the first available RTL-SDR device.
func GetFirstDevice(logger *slog.Logger) (*DeviceInfo, error) {
	devices, err := FindRTLSDRDevices(logger)
	if err != nil {
		return nil, err
	}

	if len(devices) == 0 {
		return nil, ErrNoRTLSDRDevicesFound
	}

	return &devices[0], nil
}

// FindDeviceByBusDevice finds a device by its BUS:DEV identifier.
func FindDeviceByBusDevice(busDevice string, logger *slog.Logger) (*DeviceInfo, error) {
	devices, err := FindRTLSDRDevices(logger)
	if err != nil {
		return nil, err
	}

	for _, device := range devices {
		if device.BusDevice == busDevice {
			return &device, nil
		}
	}

	return nil, fmt.Errorf("%w: %s", ErrDeviceNotFound, busDevice)
}
