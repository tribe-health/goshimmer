package connector

import (
	"github.com/iotaledger/goshimmer/dapps/valuetransfers/packages/transaction"
	"github.com/iotaledger/hive.go/events"
)

// EventValueTransactionConfirmed global event.
// Triggered whenever new confirmed transaction is confirmed
var EventValueTransactionConfirmed *events.Event

func init() {
	EventValueTransactionConfirmed = events.NewEvent(func(handler interface{}, params ...interface{}) {
		handler.(func(_ *transaction.Transaction))(params[0].(*transaction.Transaction))
	})
}
