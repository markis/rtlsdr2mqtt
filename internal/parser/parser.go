// Package parser provides functionality to parse rtlamr output.
package parser

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
)

var ErrIgnoredLine = errors.New("ignored rtlamr output line")

// Parser handles parsing rtlamr output.
type Parser struct {
	// meterIDs is a map of meter IDs we're interested in
	meterIDs map[string]bool
}

// NewParser creates a new parser instance.
func NewParser(meterIDs []string) *Parser {
	idMap := make(map[string]bool)
	for _, id := range meterIDs {
		idMap[id] = true
	}

	return &Parser{
		meterIDs: idMap,
	}
}

// ParseLine parses a single line of rtlamr output.
func (p *Parser) ParseLine(line string) (*ParsedMessage, error) {
	line = strings.TrimSpace(line)
	if line == "" {
		return nil, ErrIgnoredLine
	}

	// Skip non-JSON lines (like debug output)
	if !strings.HasPrefix(line, "{") {
		return nil, ErrIgnoredLine
	}

	var output RTLAMROutput
	if err := json.Unmarshal([]byte(line), &output); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	// Get meter ID
	meterID, err := output.GetMeterID()
	if err != nil {
		return nil, fmt.Errorf("failed to extract meter ID: %w", err)
	}

	// Check if we're interested in this meter
	if !p.meterIDs[meterID] {
		return nil, ErrIgnoredLine
	}

	// Get consumption
	consumption, err := output.GetConsumption()
	if err != nil {
		return nil, fmt.Errorf("failed to extract consumption: %w", err)
	}

	// Get attributes
	attributes, err := output.GetAttributes()
	if err != nil {
		return nil, fmt.Errorf("failed to extract attributes: %w", err)
	}

	return &ParsedMessage{
		MeterID:     meterID,
		Consumption: consumption,
		Protocol:    strings.ToLower(output.Type),
		Attributes:  attributes,
	}, nil
}

// GetMessageForIDs parses rtlamr output and returns a reading if it matches any of the meter IDs.
func GetMessageForIDs(rtlamrOutput string, meterIDsList []string) *MeterReading {
	parser := NewParser(meterIDsList)

	parsed, err := parser.ParseLine(rtlamrOutput)
	if err != nil || parsed == nil {
		return nil
	}

	return &MeterReading{
		MeterID:     parsed.MeterID,
		Consumption: parsed.Consumption,
		Message:     parsed.Attributes,
	}
}

// FormatNumber formats a number according to a format mask (e.g., "######.###").
func FormatNumber(value int64, format string) any {
	if format == "" {
		return value
	}

	// Count digits before and after decimal point
	parts := strings.Split(format, ".")
	if len(parts) != 2 {
		// No decimal point in format, return as integer
		return value
	}

	decimalPlaces := len(parts[1])
	if decimalPlaces == 0 {
		return value
	}

	// Convert to float with appropriate decimal places
	divisor := 1.0
	for range decimalPlaces {
		divisor *= 10.0
	}

	result := float64(value) / divisor

	// Format to the specified number of decimal places
	formatStr := fmt.Sprintf("%%.%df", decimalPlaces)
	formatted := fmt.Sprintf(formatStr, result)

	// Convert back to float64 for JSON
	floatResult, err := strconv.ParseFloat(formatted, 64)
	if err != nil {
		return value // Fallback to original value
	}

	return floatResult
}
