package valuetangle

import (
	"github.com/iotaledger/goshimmer/dapps/valuetransfers/packages/address"
	"github.com/iotaledger/goshimmer/dapps/valuetransfers/packages/balance"
	"github.com/iotaledger/goshimmer/dapps/valuetransfers/packages/transaction"
)

type ValueTangle interface {
	GetAddressOutputs(addr address.Address) (map[transaction.OutputID][]*balance.Balance, error)
	PostTransaction(tx *transaction.Transaction, onConfirm func()) error
	IsConfirmed(txid *transaction.ID) (bool, error)
	GetTransaction(id transaction.ID) (*transaction.Transaction, error)
	RequestFunds(target address.Address) (*transaction.Transaction, error)
}
