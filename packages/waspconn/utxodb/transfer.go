package utxodb

import (
	"fmt"
	"github.com/iotaledger/goshimmer/dapps/valuetransfers/packages/address"
	"github.com/iotaledger/goshimmer/dapps/valuetransfers/packages/balance"
	"github.com/iotaledger/goshimmer/dapps/valuetransfers/packages/transaction"
)

func DistributeIotas(amountEach int64, source address.Address, targets []address.Address) (*transaction.Transaction, error) {
	amount := amountEach * int64(len(targets))

	sourceOutputs := GetAddressOutputs(source)
	oids := make([]transaction.OutputID, 0)
	sum := int64(0)
	for oid, bals := range sourceOutputs {
		containsIotas := false
		for _, b := range bals {
			if b.Color() == balance.ColorIOTA {
				sum += b.Value()
				containsIotas = true
			}
		}
		if containsIotas {
			oids = append(oids, oid)
		}
		if sum >= amount {
			break
		}
	}
	if sum < amount {
		return nil, fmt.Errorf("not enough input balance")
	}
	inputs := transaction.NewInputs(oids...)

	out := make(map[address.Address][]*balance.Balance)
	for _, taddr := range targets {
		out[taddr] = []*balance.Balance{balance.New(balance.ColorIOTA, amountEach)}
	}
	if sum > amount {
		out[GetGenesisAddress()] = []*balance.Balance{balance.New(balance.ColorIOTA, sum-amount)}
	}

	outputs := transaction.NewOutputs(out)

	tx := transaction.New(inputs, outputs)
	if err := CheckInputsOutputs(tx); err != nil {
		panic(err)
	}

	tx.Sign(GetSigScheme(source))

	if !tx.SignaturesValid() {
		panic("something wrong with signatures")
	}
	return tx, nil
}
