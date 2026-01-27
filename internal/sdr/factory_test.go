package sdr

import (
	"log/slog"
	"os"
	"testing"

	"rtlsdr2mqtt/internal/config"
)

func TestNewSDR(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))

	tests := []struct {
		name        string
		config      *config.Config
		wantErr     bool
		expectIndex uint32
	}{
		{
			name: "default device (empty)",
			config: &config.Config{
				SDR: config.SDRConfig{
					USBDevice: "",
				},
			},
			wantErr:     false,
			expectIndex: 0,
		},
		{
			name: "numeric device ID",
			config: &config.Config{
				SDR: config.SDRConfig{
					USBDevice: "2",
				},
			},
			wantErr:     false,
			expectIndex: 2,
		},
		{
			name: "serial number (not implemented yet)",
			config: &config.Config{
				SDR: config.SDRConfig{
					USBDevice: "ABC00000001", // Non-numeric serial
				},
			},
			wantErr:     false,
			expectIndex: 0, // Falls back to 0
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			device, err := NewSDR(tt.config, logger)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewSDR() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if device == nil {
				t.Error("NewSDR() returned nil device")
				return
			}
			// Note: We can't easily test the device index without exposing it,
			// but it's set internally and used by Open()
		})
	}
}

func TestResolveDeviceIndex(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))

	tests := []struct {
		name      string
		deviceID  string
		wantIndex uint32
	}{
		{
			name:      "empty device ID",
			deviceID:  "",
			wantIndex: 0,
		},
		{
			name:      "zero device ID",
			deviceID:  "0",
			wantIndex: 0,
		},
		{
			name:      "numeric device ID",
			deviceID:  "1",
			wantIndex: 1,
		},
		{
			name:      "large numeric device ID",
			deviceID:  "99",
			wantIndex: 99,
		},
		{
			name:      "serial number",
			deviceID:  "ABC123",
			wantIndex: 0, // Falls back to 0 as serial lookup not implemented
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := resolveDeviceIndex(tt.deviceID, logger)
			if got != tt.wantIndex {
				t.Errorf("resolveDeviceIndex() = %v, want %v", got, tt.wantIndex)
			}
		})
	}
}

func TestApplyConfiguration(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))

	tests := []struct {
		name    string
		config  *config.Config
		wantErr bool
	}{
		{
			name: "default configuration",
			config: &config.Config{
				SDR: config.SDRConfig{
					FreqCorrection: 0,
					GainMode:       "auto",
					Gain:           0,
					AGCEnabled:     true,
				},
			},
			wantErr: true, // Device not open
		},
		{
			name: "manual gain mode",
			config: &config.Config{
				SDR: config.SDRConfig{
					FreqCorrection: 10,
					GainMode:       "manual",
					Gain:           496,
					AGCEnabled:     false,
				},
			},
			wantErr: true, // Device not open
		},
		{
			name: "case insensitive gain mode",
			config: &config.Config{
				SDR: config.SDRConfig{
					GainMode:   "Manual",
					AGCEnabled: true,
				},
			},
			wantErr: true, // Device not open
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			device := NewRTLSDRDevice(0)
			err := ApplyConfiguration(device, tt.config, logger)
			if (err != nil) != tt.wantErr {
				t.Errorf("ApplyConfiguration() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
