//go:build cgo && linux

// Package usb provides USB device management functions.
package usb

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
)

const usbDevfsReset = 0x5514 // Linux USB device reset ioctl

var (
	ErrInvalidDeviceFormat = errors.New("invalid device format, expected BUS:DEV")
	ErrNotCharacterDevice  = errors.New("not a character device")
	ErrUSBResetFailed      = errors.New("USB reset ioctl failed")
	ErrInvalidDevicePath   = errors.New("invalid device path")
)

// ResetUSBDevice resets a USB device by its BUS:DEV identifier.
func ResetUSBDevice(busDevice string, logger *slog.Logger) error {
	if logger == nil {
		logger = slog.Default()
	}

	logger.Debug("Resetting USB device", "device", busDevice)

	parts := strings.Split(busDevice, ":")
	if len(parts) != 2 {
		return fmt.Errorf("%w: %s", ErrInvalidDeviceFormat, busDevice)
	}

	busNum, err := strconv.Atoi(parts[0])
	if err != nil {
		return fmt.Errorf("invalid bus number '%s': %w", parts[0], err)
	}

	devNum, err := strconv.Atoi(parts[1])
	if err != nil {
		return fmt.Errorf("invalid device number '%s': %w", parts[1], err)
	}

	devicePath := fmt.Sprintf("/dev/bus/usb/%03d/%03d", busNum, devNum)

	// Validate the device path to prevent potential security issues
	cleanPath := filepath.Clean(devicePath)
	if !strings.HasPrefix(cleanPath, "/dev/bus/usb/") {
		return fmt.Errorf("%w: %s", ErrInvalidDevicePath, cleanPath)
	}

	// Check if device exists and is a character device
	stat, err := os.Stat(cleanPath)
	if err != nil {
		return fmt.Errorf("device not found at %s: %w", cleanPath, err)
	}

	mode := stat.Mode()
	if mode&os.ModeCharDevice == 0 {
		return fmt.Errorf("%w: %s", ErrNotCharacterDevice, cleanPath)
	}

	// Open the device file
	file, err := os.OpenFile(cleanPath, os.O_WRONLY, 0)
	if err != nil {
		return fmt.Errorf("failed to open device %s: %w", cleanPath, err)
	}
	defer file.Close()

	// Perform the USB reset ioctl
	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, file.Fd(), usbDevfsReset, 0)
	if errno != 0 {
		return fmt.Errorf("%w: %s", ErrUSBResetFailed, errno.Error())
	}

	logger.Info("USB device reset successful", "device", busDevice, "path", cleanPath)
	return nil
}

// ResetFirstDevice resets the first available RTL-SDR device.
func ResetFirstDevice(logger *slog.Logger) error {
	device, err := GetFirstDevice(logger)
	if err != nil {
		return fmt.Errorf("failed to find device to reset: %w", err)
	}

	return ResetUSBDevice(device.BusDevice, logger)
}

// ResetDeviceByID resets a device by its vendor:product ID.
func ResetDeviceByID(deviceID string, logger *slog.Logger) error {
	devices, err := FindRTLSDRDevices(logger)
	if err != nil {
		return fmt.Errorf("failed to find devices: %w", err)
	}

	// If deviceID is "0", use the first device
	if deviceID == "0" {
		if len(devices) == 0 {
			return ErrNoRTLSDRDevicesFound
		}
		return ResetUSBDevice(devices[0].BusDevice, logger)
	}

	// Look for device with matching bus:device or vendor:product
	for _, device := range devices {
		if device.BusDevice == deviceID {
			return ResetUSBDevice(device.BusDevice, logger)
		}

		vendorProduct := fmt.Sprintf("%04x:%04x", device.VendorID, device.ProductID)
		if strings.EqualFold(vendorProduct, deviceID) {
			return ResetUSBDevice(device.BusDevice, logger)
		}
	}

	return fmt.Errorf("%w: %s", ErrDeviceNotFound, deviceID)
}
