package drng

import (
	"github.com/iotaledger/goshimmer/packages/binary/drng/state"
	cbEvents "github.com/iotaledger/goshimmer/packages/binary/drng/subtypes/collectivebeacon/events"
	"github.com/iotaledger/hive.go/events"
)

// DRNG holds the state and events of a drng instance.
type DRNG struct {
	State  map[uint32]*state.State // The state of the DRNG.
	Events *Event                  // The events fired on the DRNG.
}

// New creates a new DRNG instance.
func New(config map[uint32][]state.Option) *DRNG {
	drng := &DRNG{
		State: make(map[uint32]*state.State),
		Events: &Event{
			CollectiveBeacon: events.NewEvent(cbEvents.CollectiveBeaconReceived),
			Randomness:       events.NewEvent(randomnessReceived),
		},
	}
	if len(config) > 0 {
		for id, setters := range config {
			drng.State[id] = state.New(setters...)
		}
	}
	return drng
}
