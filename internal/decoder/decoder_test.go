package decoder

import (
	"log/slog"
	"os"
	"slices"
	"testing"

	"rtlsdr2mqtt/internal/config"
)

func TestNewDecoder(t *testing.T) {
	cfg := &config.Config{
		SDR: config.SDRConfig{
			USBDevice: "",
		},
		Meters: []config.MeterConfig{
			{
				ID:       "12345678",
				Protocol: "scm+",
			},
		},
	}

	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))

	decoder := NewDecoder(cfg, logger)

	if decoder == nil {
		t.Fatal("NewDecoder returned nil")
	}

	if decoder.config != cfg {
		t.Error("Decoder config not set correctly")
	}

	if decoder.logger != logger {
		t.Error("Decoder logger not set correctly")
	}

	if decoder.msgChan == nil {
		t.Error("Message channel not initialized")
	}

	if decoder.errChan == nil {
		t.Error("Error channel not initialized")
	}

	if decoder.doneChan == nil {
		t.Error("Done channel not initialized")
	}

	if decoder.isRunning.Load() {
		t.Error("Decoder should not be running initially")
	}
}

func TestNewDecoderNilLogger(t *testing.T) {
	cfg := &config.Config{
		Meters: []config.MeterConfig{
			{ID: "12345", Protocol: "scm+"},
		},
	}

	decoder := NewDecoder(cfg, nil)

	if decoder == nil {
		t.Fatal("NewDecoder returned nil")
	}

	if decoder.logger == nil {
		t.Error("Decoder should have default logger when nil is passed")
	}
}

func TestIsRunning(t *testing.T) {
	cfg := &config.Config{
		Meters: []config.MeterConfig{
			{ID: "12345", Protocol: "scm+"},
		},
	}

	decoder := NewDecoder(cfg, nil)

	if decoder.IsRunning() {
		t.Error("Decoder should not be running initially")
	}
}

func TestGetProtocols(t *testing.T) {
	tests := []struct {
		name              string
		meters            []config.MeterConfig
		expectedProtocols []string
	}{
		{
			name: "single protocol",
			meters: []config.MeterConfig{
				{ID: "12345", Protocol: "scm+"},
			},
			expectedProtocols: []string{"scm+"},
		},
		{
			name: "multiple different protocols",
			meters: []config.MeterConfig{
				{ID: "12345", Protocol: "scm+"},
				{ID: "67890", Protocol: "idm"},
				{ID: "11111", Protocol: "r900"},
			},
			expectedProtocols: []string{"scm+", "idm", "r900"},
		},
		{
			name: "duplicate protocols",
			meters: []config.MeterConfig{
				{ID: "12345", Protocol: "scm+"},
				{ID: "67890", Protocol: "scm+"},
				{ID: "11111", Protocol: "idm"},
			},
			expectedProtocols: []string{"scm+", "idm"},
		},
		{
			name:              "no meters - should return defaults",
			meters:            []config.MeterConfig{},
			expectedProtocols: []string{"scm", "scm+", "idm", "r900"},
		},
		{
			name: "mixed case protocols",
			meters: []config.MeterConfig{
				{ID: "12345", Protocol: "SCM+"},
				{ID: "67890", Protocol: "IdM"},
			},
			expectedProtocols: []string{"scm+", "idm"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{Meters: tt.meters}
			decoder := NewDecoder(cfg, nil)

			protocols := decoder.getProtocols()

			// Check length
			if len(protocols) != len(tt.expectedProtocols) {
				t.Errorf("Expected %d protocols, got %d: %v", len(tt.expectedProtocols), len(protocols), protocols)
			}

			// Check all expected protocols are present
			for _, expected := range tt.expectedProtocols {
				if !slices.Contains(protocols, expected) {
					t.Errorf("Expected protocol %s not found in result: %v", expected, protocols)
				}
			}
		})
	}
}

func TestGetMeterIDs(t *testing.T) {
	tests := []struct {
		name        string
		meters      []config.MeterConfig
		expectedIDs []uint32
	}{
		{
			name: "single meter",
			meters: []config.MeterConfig{
				{ID: "12345", Protocol: "scm+"},
			},
			expectedIDs: []uint32{12345},
		},
		{
			name: "multiple meters",
			meters: []config.MeterConfig{
				{ID: "12345", Protocol: "scm+"},
				{ID: "67890", Protocol: "idm"},
				{ID: "11111", Protocol: "r900"},
			},
			expectedIDs: []uint32{12345, 67890, 11111},
		},
		{
			name:        "no meters",
			meters:      []config.MeterConfig{},
			expectedIDs: []uint32{},
		},
		{
			name: "invalid meter ID",
			meters: []config.MeterConfig{
				{ID: "12345", Protocol: "scm+"},
				{ID: "invalid", Protocol: "idm"},
				{ID: "67890", Protocol: "r900"},
			},
			expectedIDs: []uint32{12345, 67890},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{Meters: tt.meters}
			decoder := NewDecoder(cfg, nil)

			ids := decoder.getMeterIDs()

			if len(ids) != len(tt.expectedIDs) {
				t.Errorf("Expected %d IDs, got %d: %v", len(tt.expectedIDs), len(ids), ids)
			}

			for i, expected := range tt.expectedIDs {
				if i >= len(ids) {
					t.Errorf("Missing expected ID %d", expected)
					continue
				}
				if ids[i] != expected {
					t.Errorf("Expected ID %d, got %d", expected, ids[i])
				}
			}
		})
	}
}

func TestMessageHelperMethods(t *testing.T) {
	msg := &Message{
		MeterID:     87654321,
		Consumption: 123456,
		Protocol:    "scm+",
		Attributes:  map[string]any{"test": "value"},
	}

	// Test MeterIDString
	idStr := msg.MeterIDString()
	if idStr != "87654321" {
		t.Errorf("Expected MeterIDString '87654321', got '%s'", idStr)
	}

	// Test ConsumptionInt64
	consumption := msg.ConsumptionInt64()
	if consumption != 123456 {
		t.Errorf("Expected ConsumptionInt64 123456, got %d", consumption)
	}
}

func TestDecoderErrors(t *testing.T) {
	// Test error constants exist
	if ErrDecoderNotStarted == nil {
		t.Error("ErrDecoderNotStarted should be defined")
	}

	if ErrDecoderStopped == nil {
		t.Error("ErrDecoderStopped should be defined")
	}

	if ErrDecoderTimeout == nil {
		t.Error("ErrDecoderTimeout should be defined")
	}

	if ErrShortRead == nil {
		t.Error("ErrShortRead should be defined")
	}

	// Test error messages
	if ErrDecoderNotStarted.Error() != "decoder is not started" {
		t.Errorf("ErrDecoderNotStarted has wrong message: %v", ErrDecoderNotStarted)
	}

	if ErrDecoderStopped.Error() != "decoder has been stopped" {
		t.Errorf("ErrDecoderStopped has wrong message: %v", ErrDecoderStopped)
	}

	if ErrDecoderTimeout.Error() != "decoder timeout while reading messages" {
		t.Errorf("ErrDecoderTimeout has wrong message: %v", ErrDecoderTimeout)
	}

	if ErrShortRead.Error() != "short read from SDR device" {
		t.Errorf("ErrShortRead has wrong message: %v", ErrShortRead)
	}
}
