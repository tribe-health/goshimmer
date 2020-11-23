package pow

import (
	"time"

	"github.com/iotaledger/hive.go/events"
)

type PowEvents struct {
	// PowDone defines the pow done event.
	PowDone *events.Event
}

// PowDoneEvent is used to pass information through a PowDone event.
type PowDoneEvent struct {
	Difficulty int
	Duration   time.Duration
}

func powDoneEventCaller(handler interface{}, params ...interface{}) {
	handler.(func(ev *PowDoneEvent))(params[0].(*PowDoneEvent))
}
