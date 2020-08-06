package valuetangle

import (
	"fmt"

	"github.com/iotaledger/goshimmer/dapps/faucet"
	faucetpayload "github.com/iotaledger/goshimmer/dapps/faucet/packages/payload"
	"github.com/iotaledger/goshimmer/dapps/valuetransfers"
	"github.com/iotaledger/goshimmer/dapps/valuetransfers/packages/address"
	"github.com/iotaledger/goshimmer/dapps/valuetransfers/packages/balance"
	"github.com/iotaledger/goshimmer/dapps/valuetransfers/packages/tangle"
	"github.com/iotaledger/goshimmer/dapps/valuetransfers/packages/transaction"
	"github.com/iotaledger/goshimmer/plugins/config"
	"github.com/iotaledger/goshimmer/plugins/issuer"
	"github.com/iotaledger/goshimmer/plugins/messagelayer"
	"github.com/iotaledger/hive.go/events"
)

// interface between waspconn and the value tangle
type ValueTangle interface {
	GetAddressOutputs(addr address.Address) (map[transaction.OutputID][]*balance.Balance, error)
	PostTransaction(tx *transaction.Transaction) error
	OnTransactionConfirmed(func(tx *transaction.Transaction))
	IsConfirmed(txid *transaction.ID) (bool, error)
	GetTransaction(txid *transaction.ID) *transaction.Transaction
	RequestFunds(target address.Address) error
	Detach()
}

type valuetangle struct {
	txConfirmedClosure  *events.Closure
	txConfirmedCallback func(tx *transaction.Transaction)
}

func NewRealValueTangle() *valuetangle {
	v := &valuetangle{}

	v.txConfirmedClosure = events.NewClosure(func(ctx *transaction.CachedTransaction, ctxMeta *tangle.CachedTransactionMetadata) {
		tx := ctx.Unwrap()
		if tx != nil && v.txConfirmedCallback != nil {
			v.txConfirmedCallback(tx)
		}
	})
	valuetransfers.Tangle().Events.TransactionConfirmed.Attach(v.txConfirmedClosure)

	return v
}

func (v *valuetangle) Detach() {
	valuetransfers.Tangle().Events.TransactionConfirmed.Detach(v.txConfirmedClosure)
}

func (v *valuetangle) OnTransactionConfirmed(cb func(tx *transaction.Transaction)) {
	v.txConfirmedCallback = cb
}

func (v *valuetangle) GetAddressOutputs(addr address.Address) (map[transaction.OutputID][]*balance.Balance, error) {
	ret := make(map[transaction.OutputID][]*balance.Balance)
	valuetransfers.Tangle().OutputsOnAddress(addr).Consume(func(output *tangle.Output) {
		if output.ConsumerCount() == 0 {
			ret[output.ID()] = output.Balances()
		}
	})
	return ret, nil
}

func (v *valuetangle) PostTransaction(tx *transaction.Transaction) error {
	// prepare value payload with value factory
	payload, err := valuetransfers.ValueObjectFactory().IssueTransaction(tx)
	if err != nil {
		return fmt.Errorf("failed to issue transaction: %w", err)
	}

	// attach to message layer
	_, err = issuer.IssuePayload(payload)
	return err
}

func (v *valuetangle) IsConfirmed(txid *transaction.ID) (bool, error) {
	cachedTxnMetaObj := valuetransfers.Tangle().TransactionMetadata(*txid)
	defer cachedTxnMetaObj.Release()
	if !cachedTxnMetaObj.Exists() {
		return false, fmt.Errorf("Transaction not found")
	}
	return cachedTxnMetaObj.Unwrap().Confirmed(), nil
}

func (v *valuetangle) GetTransaction(txid *transaction.ID) *transaction.Transaction {
	cachedTxnObj := valuetransfers.Tangle().Transaction(*txid)
	defer cachedTxnObj.Release()
	if !cachedTxnObj.Exists() {
		return nil
	}
	return cachedTxnObj.Unwrap()
}

func (v *valuetangle) RequestFunds(target address.Address) error {
	faucetPayload, err := faucetpayload.New(target, config.Node().GetInt(faucet.CfgFaucetPoWDifficulty))
	if err != nil {
		return err
	}
	_, err = messagelayer.MessageFactory().IssuePayload(faucetPayload)
	return err
}
