// Package sdr provides a USB-enumeration device source shared by startup and
// re-enumeration recovery.
package sdr

import (
	"fmt"
	"log/slog"

	"rtlsdr2mqtt/internal/config"
)

// ConfigDeviceSource acquires RTL-SDR devices through fresh USB enumeration
// using the configuration's device-selection semantics. Both the startup
// path and the re-enumeration recovery path use it, so recovery opens the
// same logical device startup would choose: the configured device, or the
// first available RTL-SDR when none is configured. A re-enumerated dongle
// may appear at a different bus/device node; selection never assumes the
// prior node path.
type ConfigDeviceSource struct {
	cfg       *config.Config
	logger    *slog.Logger
	newDevice func(index uint32) SDR
}

// NewDeviceSource creates the production device source for the given config.
func NewDeviceSource(cfg *config.Config, logger *slog.Logger) *ConfigDeviceSource {
	if logger == nil {
		logger = slog.Default()
	}
	return &ConfigDeviceSource{
		cfg:       cfg,
		logger:    logger,
		newDevice: func(index uint32) SDR { return NewRTLSDRDevice(index) },
	}
}

// Acquire enumerates the USB bus for a matching RTL-SDR, opens the selected
// device, and returns it with a short identity string for logs. It returns
// ErrDeviceAbsent when no matching device is currently enumerated; opening a
// device that was just enumerated (busy, lost in a re-enumeration race) is
// returned as a plain open error so callers retry through the same backoff
// path.
func (s *ConfigDeviceSource) Acquire() (SDR, string, error) {
	index := resolveDeviceIndex(s.cfg.SDR.USBDevice, s.logger)

	device := s.newDevice(index)

	// Probe via fresh librtlsdr enumeration (VID:PID based): this is both the
	// presence check for an absent dongle and the re-enumeration poll that
	// notices a replugged dongle at a new bus/device node.
	if count := device.GetDeviceCount(); count <= index {
		return nil, "", fmt.Errorf("%w: selected index %d with %d device(s) enumerated",
			ErrDeviceAbsent, index, count)
	}

	if err := device.Open(); err != nil {
		return nil, "", fmt.Errorf("failed to open RTL-SDR device: %w", err)
	}

	identity := fmt.Sprintf("%s (index %d)", device.GetDeviceName(index), index)
	s.logger.Info("RTL-SDR device acquired", "device", identity)
	return device, identity, nil
}
