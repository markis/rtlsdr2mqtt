package parser

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
)

var (
	ErrUnsupportedMessageType = errors.New("unsupported message type")
	ErrConsumptionValueTooBig = errors.New("consumption value too large")
)

// RTLAMROutput represents the JSON output from rtlamr.
type RTLAMROutput struct {
	Time    string          `json:"Time"`
	Offset  int             `json:"Offset"`
	Length  int             `json:"Length"`
	Type    string          `json:"Type"` // SCM, SCM+, IDM, NetIDM, R900, R900BCD
	Message json.RawMessage `json:"Message"`
}

// SCMMessage represents a Standard Consumption Message.
type SCMMessage struct {
	ID           uint32 `json:"ID"`         // Legacy field name
	EndpointID   uint32 `json:"EndpointID"` // New field name in rtlamr v0.9.4+
	Type         int    `json:"Type"`
	ProtocolID   int    `json:"ProtocolID"`   // New field in v0.9.4+
	EndpointType int    `json:"EndpointType"` // New field in v0.9.4+
	TamperPhy    int    `json:"TamperPhy"`
	TamperEnc    int    `json:"TamperEnc"`
	Tamper       int    `json:"Tamper"` // New tamper field format
	Consumption  uint64 `json:"Consumption"`
	ChecksumVal  uint32 `json:"ChecksumVal"`
	PacketCRC    uint32 `json:"PacketCRC"` // New field in v0.9.4+
	FrameSync    uint32 `json:"FrameSync"` // New field in v0.9.4+
}

// IDMMessage represents an Interval Data Message.
type IDMMessage struct {
	Preamble                         uint32 `json:"Preamble"`
	PacketTypeID                     int    `json:"PacketTypeID"`
	PacketLength                     int    `json:"PacketLength"`
	HammingCode                      int    `json:"HammingCode"`
	ApplicationVersion               int    `json:"ApplicationVersion"`
	ERTType                          int    `json:"ERTType"`
	ERTSerialNumber                  uint32 `json:"ERTSerialNumber"`
	ConsumptionIntervalCount         int    `json:"ConsumptionIntervalCount"`
	ModuleProgrammingState           int    `json:"ModuleProgrammingState"`
	TamperCounters                   string `json:"TamperCounters"`
	AsynchronousCounters             int    `json:"AsynchronousCounters"`
	PowerOutageFlags                 string `json:"PowerOutageFlags"`
	LastConsumptionCount             uint64 `json:"LastConsumptionCount"`
	DifferentialConsumptionIntervals []int  `json:"DifferentialConsumptionIntervals"`
	TransmitTimeOffset               int    `json:"TransmitTimeOffset"`
	SerialNumberCRC                  uint32 `json:"SerialNumberCRC"`
	PacketCRC                        uint32 `json:"PacketCRC"`
}

// R900Message represents an R900 protocol message.
type R900Message struct {
	ID          uint64 `json:"ID"`
	Unkn1       int    `json:"Unkn1"`
	NoUse       int    `json:"NoUse"`
	BackFlow    int    `json:"BackFlow"`
	Consumption uint64 `json:"Consumption"`
	Unkn3       int    `json:"Unkn3"`
	Leak        int    `json:"Leak"`
	LeakNow     int    `json:"LeakNow"`
}

// MeterReading represents a parsed meter reading.
type MeterReading struct {
	MeterID     string         `json:"meter_id"`
	Consumption int64          `json:"consumption"`
	Message     map[string]any `json:"message"` // Raw message fields for attributes
}

// ParsedMessage represents a parsed message with extracted information.
type ParsedMessage struct {
	MeterID     string
	Consumption int64
	Protocol    string
	Attributes  map[string]any
}

// GetMeterID extracts the meter ID from a message based on protocol.
func (msg *RTLAMROutput) GetMeterID() (string, error) {
	switch msg.Type {
	case "SCM", "SCM+", "R900", "R900BCD":
		var scmMsg SCMMessage
		if err := json.Unmarshal(msg.Message, &scmMsg); err != nil {
			return "", fmt.Errorf("failed to parse %s message: %w", msg.Type, err)
		}
		// Handle both legacy ID field and new EndpointID field
		if scmMsg.EndpointID != 0 {
			return strconv.FormatUint(uint64(scmMsg.EndpointID), 10), nil
		}
		return strconv.FormatUint(uint64(scmMsg.ID), 10), nil

	case "IDM", "NetIDM":
		var idmMsg IDMMessage
		if err := json.Unmarshal(msg.Message, &idmMsg); err != nil {
			return "", fmt.Errorf("failed to parse %s message: %w", msg.Type, err)
		}
		return strconv.FormatUint(uint64(idmMsg.ERTSerialNumber), 10), nil

	default:
		return "", fmt.Errorf("%w: %s", ErrUnsupportedMessageType, msg.Type)
	}
}

// GetConsumption extracts the consumption value from a message based on protocol.
func (msg *RTLAMROutput) GetConsumption() (int64, error) {
	switch msg.Type {
	case "SCM", "SCM+":
		var scmMsg SCMMessage
		if err := json.Unmarshal(msg.Message, &scmMsg); err != nil {
			return 0, fmt.Errorf("failed to parse %s message: %w", msg.Type, err)
		}
		if scmMsg.Consumption > math.MaxInt64 {
			return 0, fmt.Errorf("%w: %d", ErrConsumptionValueTooBig, scmMsg.Consumption)
		}
		return int64(scmMsg.Consumption), nil

	case "R900", "R900BCD":
		var r900Msg R900Message
		if err := json.Unmarshal(msg.Message, &r900Msg); err != nil {
			return 0, fmt.Errorf("failed to parse %s message: %w", msg.Type, err)
		}
		if r900Msg.Consumption > math.MaxInt64 {
			return 0, fmt.Errorf("%w: %d", ErrConsumptionValueTooBig, r900Msg.Consumption)
		}
		return int64(r900Msg.Consumption), nil

	case "IDM", "NetIDM":
		var idmMsg IDMMessage
		if err := json.Unmarshal(msg.Message, &idmMsg); err != nil {
			return 0, fmt.Errorf("failed to parse %s message: %w", msg.Type, err)
		}
		if idmMsg.LastConsumptionCount > math.MaxInt64 {
			return 0, fmt.Errorf("%w: %d", ErrConsumptionValueTooBig, idmMsg.LastConsumptionCount)
		}
		return int64(idmMsg.LastConsumptionCount), nil

	default:
		return 0, fmt.Errorf("%w: %s", ErrUnsupportedMessageType, msg.Type)
	}
}

// GetAttributes returns all message fields as a map for attributes topic.
func (msg *RTLAMROutput) GetAttributes() (map[string]any, error) {
	var attributes map[string]any
	if err := json.Unmarshal(msg.Message, &attributes); err != nil {
		return nil, fmt.Errorf("failed to parse message attributes: %w", err)
	}

	// Add protocol type to attributes
	attributes["protocol"] = strings.ToLower(msg.Type)

	return attributes, nil
}
