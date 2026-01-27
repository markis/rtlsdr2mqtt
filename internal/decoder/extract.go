package decoder

import (
	"fmt"

	"github.com/bemasher/rtlamr/idm"
	"github.com/bemasher/rtlamr/netidm"
	"github.com/bemasher/rtlamr/protocol"
	"github.com/bemasher/rtlamr/r900"
	"github.com/bemasher/rtlamr/r900bcd"
	"github.com/bemasher/rtlamr/scm"
	"github.com/bemasher/rtlamr/scmplus"
)

// extractMessageData extracts consumption value and attributes from a protocol message.
// Returns the consumption value and a map of all message attributes.
func extractMessageData(msg protocol.Message) (uint32, map[string]any) {
	attrs := make(map[string]any)

	// Add common attributes
	attrs["protocol"] = msg.MsgType()
	attrs["meter_type"] = msg.MeterType()

	switch m := msg.(type) {
	case scm.SCM:
		attrs["id"] = m.ID
		attrs["type"] = m.Type
		attrs["tamper_phy"] = m.TamperPhy
		attrs["tamper_enc"] = m.TamperEnc
		attrs["checksum"] = fmt.Sprintf("0x%04X", m.ChecksumVal)
		return m.Consumption, attrs

	case scmplus.SCM:
		attrs["endpoint_id"] = m.EndpointID
		attrs["endpoint_type"] = m.EndpointType
		attrs["protocol_id"] = m.ProtocolID
		attrs["tamper"] = m.Tamper
		attrs["checksum"] = fmt.Sprintf("0x%04X", m.PacketCRC)
		return m.Consumption, attrs

	case r900.R900:
		attrs["id"] = m.ID
		attrs["unkn1"] = m.Unkn1
		attrs["no_use"] = m.NoUse
		attrs["back_flow"] = m.BackFlow
		attrs["unkn3"] = m.Unkn3
		attrs["leak"] = m.Leak
		attrs["leak_now"] = m.LeakNow
		return m.Consumption, attrs

	case r900bcd.R900BCD:
		attrs["id"] = m.ID
		attrs["unkn1"] = m.Unkn1
		attrs["no_use"] = m.NoUse
		attrs["back_flow"] = m.BackFlow
		attrs["unkn3"] = m.Unkn3
		attrs["leak"] = m.Leak
		attrs["leak_now"] = m.LeakNow
		return m.Consumption, attrs

	case idm.IDM:
		attrs["ert_serial_number"] = m.ERTSerialNumber
		attrs["ert_type"] = m.ERTType
		attrs["consumption_interval_count"] = m.ConsumptionIntervalCount
		attrs["module_programming_state"] = m.ModuleProgrammingState
		attrs["tamper_counters"] = fmt.Sprintf("%02X", m.TamperCounters)
		attrs["asynchronous_counters"] = m.AsynchronousCounters
		attrs["power_outage_flags"] = fmt.Sprintf("%02X", m.PowerOutageFlags)
		attrs["transmit_time_offset"] = m.TransmitTimeOffset
		attrs["checksum"] = fmt.Sprintf("0x%04X", m.PacketCRC)
		// IDM uses LastConsumptionCount for the consumption value
		return m.LastConsumptionCount, attrs

	case netidm.NetIDM:
		attrs["ert_serial_number"] = m.ERTSerialNumber
		attrs["ert_type"] = m.ERTType
		attrs["consumption_interval_count"] = m.ConsumptionIntervalCount
		attrs["programming_state"] = m.ProgrammingState
		attrs["last_generation"] = m.LastGeneration
		attrs["last_consumption"] = m.LastConsumption
		attrs["last_consumption_net"] = m.LastConsumptionNet
		attrs["transmit_time_offset"] = m.TransmitTimeOffset
		attrs["checksum"] = fmt.Sprintf("0x%04X", m.PacketCRC)
		// NetIDM uses LastConsumption for the consumption value
		return m.LastConsumption, attrs

	default:
		// For unknown message types, try to extract basic info
		return 0, attrs
	}
}
