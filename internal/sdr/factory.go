// Package sdr provides a factory for creating SDR instances.
package sdr

import (
	"fmt"
	"log/slog"
	"strconv"
	"strings"

	"rtlsdr2mqtt/internal/config"
)

// NewSDR creates an appropriate SDR instance based on the configuration.
// This uses direct RTL-SDR access via librtlsdr.
func NewSDR(cfg *config.Config, logger *slog.Logger) (SDR, error) {
	if logger == nil {
		logger = slog.Default()
	}

	// Resolve device index from device ID
	deviceIndex := resolveDeviceIndex(cfg.SDR.USBDevice, logger)

	// Create device with the resolved index
	device := NewRTLSDRDevice(deviceIndex)

	logger.Info("Using direct RTL-SDR USB access", "device_index", deviceIndex)

	return device, nil
}

// resolveDeviceIndex converts a device ID (serial number or index) to a device index.
func resolveDeviceIndex(deviceID string, logger *slog.Logger) uint32 {
	// If empty, use first device
	if deviceID == "" {
		logger.Info("Using first available RTL-SDR device", "index", 0)
		return 0
	}

	// Try to parse as device index
	if index, err := strconv.ParseUint(deviceID, 10, 32); err == nil {
		logger.Info("Using RTL-SDR device by index", "index", index)
		return uint32(index)
	}

	// Otherwise, treat as serial number (will be implemented when device is opened)
	// For now, log a warning and use device 0
	logger.Warn("Serial number lookup not yet implemented, using first device",
		"requested_serial", deviceID, "using_index", 0)
	return 0
}

// ApplyConfiguration applies all configuration settings to the SDR device.
func ApplyConfiguration(sdr SDR, cfg *config.Config, logger *slog.Logger) error {
	// Set frequency correction (PPM)
	if cfg.SDR.FreqCorrection != 0 {
		if err := sdr.SetFreqCorrection(cfg.SDR.FreqCorrection); err != nil {
			return fmt.Errorf("failed to set frequency correction: %w", err)
		}
		logger.Info("Set frequency correction", "ppm", cfg.SDR.FreqCorrection)
	}

	// Set gain mode
	manualGain := strings.EqualFold(cfg.SDR.GainMode, "manual")
	if err := sdr.SetGainMode(manualGain); err != nil {
		return fmt.Errorf("failed to set gain mode: %w", err)
	}

	if manualGain {
		// Set manual gain value
		if err := sdr.SetGain(cfg.SDR.Gain); err != nil {
			return fmt.Errorf("failed to set gain: %w", err)
		}
		logger.Info("Set manual gain", "gain_db", float64(cfg.SDR.Gain)/10.0)
	} else {
		logger.Info("Using automatic gain control")
	}

	// Set AGC mode
	if cfg.SDR.AGCEnabled {
		if err := sdr.SetAGCMode(true); err != nil {
			return fmt.Errorf("failed to enable AGC: %w", err)
		}
		logger.Info("AGC enabled")
	}

	return nil
}
