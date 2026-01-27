package decoder

import "github.com/bemasher/rtlamr/protocol"

// messageFilter is an interface for filtering messages.
type messageFilter interface {
	filter(msg protocol.Message) bool
}

// filterChain holds a list of filters to apply to messages.
type filterChain []messageFilter

// add adds a filter to the chain.
func (fc *filterChain) add(filter messageFilter) {
	*fc = append(*fc, filter)
}

// match returns true if the message passes all filters.
func (fc filterChain) match(msg protocol.Message) bool {
	if len(fc) == 0 {
		return true
	}

	for _, filter := range fc {
		if !filter.filter(msg) {
			return false
		}
	}

	return true
}

// meterIDFilter filters messages by meter ID.
type meterIDFilter struct {
	ids map[uint32]bool
}

// newMeterIDFilter creates a new meter ID filter.
func newMeterIDFilter(ids []uint32) *meterIDFilter {
	idMap := make(map[uint32]bool, len(ids))
	for _, id := range ids {
		idMap[id] = true
	}
	return &meterIDFilter{ids: idMap}
}

// filter returns true if the message's meter ID is in the allowed list.
func (f *meterIDFilter) filter(msg protocol.Message) bool {
	return f.ids[msg.MeterID()]
}
