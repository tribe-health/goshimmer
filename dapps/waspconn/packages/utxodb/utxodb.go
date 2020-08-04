package utxodb

import (
	"errors"
	"fmt"
	"github.com/iotaledger/goshimmer/dapps/valuetransfers/packages/address"
	"github.com/iotaledger/goshimmer/dapps/valuetransfers/packages/balance"
	"github.com/iotaledger/goshimmer/dapps/valuetransfers/packages/transaction"
	"sync"
)

type UtxoDB struct {
	transactions  map[transaction.ID]*transaction.Transaction
	utxo          map[transaction.OutputID]bool
	utxoByAddress map[address.Address][]transaction.ID
	mutex         *sync.RWMutex
	genesisTxId   transaction.ID
}

func New() *UtxoDB {
	u := &UtxoDB{
		transactions:  make(map[transaction.ID]*transaction.Transaction),
		utxo:          make(map[transaction.OutputID]bool),
		utxoByAddress: make(map[address.Address][]transaction.ID),
		mutex:         &sync.RWMutex{},
	}
	u.genesisInit()
	return u
}

func (u *UtxoDB) ValidateTransaction(tx *transaction.Transaction) error {
	if err := u.CheckInputsOutputs(tx); err != nil {
		return fmt.Errorf("%v: txid %s", err, tx.ID().String())
	}
	if !tx.SignaturesValid() {
		return fmt.Errorf("invalid signature txid = %s", tx.ID().String())
	}
	return nil
}

func AreConflicting(tx1, tx2 *transaction.Transaction) bool {
	if tx1.ID() == tx2.ID() {
		return true
	}
	ret := false
	tx1.Inputs().ForEach(func(oid1 transaction.OutputID) bool {
		tx2.Inputs().ForEach(func(oid2 transaction.OutputID) bool {
			if oid1 == oid2 {
				ret = true
				return false
			}
			return true
		})
		return true
	})
	return ret
}

func (u *UtxoDB) IsConfirmed(txid *transaction.ID) bool {
	u.mutex.Lock()
	defer u.mutex.Unlock()
	_, ok := u.transactions[*txid]
	return ok
}

func (u *UtxoDB) AddTransaction(tx *transaction.Transaction) error {
	//fmt.Printf("[utxodb] AddTransaction::\n%s\n", tx.String())
	if err := u.ValidateTransaction(tx); err != nil {
		return err
	}

	u.mutex.Lock()
	defer u.mutex.Unlock()

	if _, ok := u.transactions[tx.ID()]; ok {
		return fmt.Errorf("duplicate transaction txid = %s", tx.ID().String())
	}

	var err error

	// check if outputs exist
	tx.Inputs().ForEach(func(outputId transaction.OutputID) bool {
		if _, ok := u.utxo[outputId]; !ok {
			err = fmt.Errorf("output doesn't exist txid = %s", tx.ID().String())
			return true
		}
		return false
	})
	if err != nil {
		return fmt.Errorf("conflict/double spend: '%v' txid %s", err, tx.ID().String())
	}

	// add outputs to utxo ledger
	// delete inputs from utxo ledger
	tx.Inputs().ForEach(func(outputId transaction.OutputID) bool {
		delete(u.utxo, outputId)
		lst, ok := u.utxoByAddress[outputId.Address()]
		if ok {
			newLst := make([]transaction.ID, 0, len(lst))
			for _, txid := range lst {
				if txid != outputId.TransactionID() {
					newLst = append(newLst, txid)
				}
			}
			u.utxoByAddress[outputId.Address()] = newLst
		}
		return true
	})

	tx.Outputs().ForEach(func(addr address.Address, bals []*balance.Balance) bool {
		u.utxo[transaction.NewOutputID(addr, tx.ID())] = true
		lst, ok := u.utxoByAddress[addr]
		if !ok {
			lst = make([]transaction.ID, 0)
		}
		lst = append(lst, tx.ID())
		u.utxoByAddress[addr] = lst
		return true
	})
	u.transactions[tx.ID()] = tx
	u.checkLedgerBalance()
	return nil
}

func (u *UtxoDB) GetTransaction(id transaction.ID) (*transaction.Transaction, bool) {
	u.mutex.RLock()
	defer u.mutex.RUnlock()

	return u.getTransaction(id)
}

func (u *UtxoDB) getTransaction(id transaction.ID) (*transaction.Transaction, bool) {
	tx, ok := u.transactions[id]
	return tx, ok
}

func (u *UtxoDB) mustGetTransaction(id transaction.ID) *transaction.Transaction {
	tx, ok := u.transactions[id]
	if !ok {
		panic(fmt.Sprintf("tx id doesn't exist: %s", id.String()))
	}
	return tx
}

func (u *UtxoDB) MustGetTransaction(id transaction.ID) *transaction.Transaction {
	u.mutex.RLock()
	defer u.mutex.RUnlock()
	return u.mustGetTransaction(id)
}

func (u *UtxoDB) GetAddressOutputs(addr address.Address) map[transaction.OutputID][]*balance.Balance {
	u.mutex.RLock()
	defer u.mutex.RUnlock()

	return u.getAddressOutputs(addr)
}

func (u *UtxoDB) getAddressOutputs(addr address.Address) map[transaction.OutputID][]*balance.Balance {
	ret := make(map[transaction.OutputID][]*balance.Balance)

	txIds, ok := u.utxoByAddress[addr]
	if !ok || len(txIds) == 0 {
		return nil
	}
	var nilid transaction.ID
	for _, txid := range txIds {
		if txid == nilid {
			panic("txid == nilid")
		}
		txInp := u.mustGetTransaction(txid)
		bals, ok := txInp.Outputs().Get(addr)
		if !ok {
			panic("output does not exist")
		}
		// adjust to new_color
		balsAdjusted := make([]*balance.Balance, len(bals.([]*balance.Balance)))
		for i, bal := range bals.([]*balance.Balance) {
			col := bal.Color
			if col == balance.ColorNew {
				col = (balance.Color)(txInp.ID())
			}
			balsAdjusted[i] = balance.New(col, bal.Value)
		}
		ret[transaction.NewOutputID(addr, txid)] = balsAdjusted
	}
	return ret
}

func (u *UtxoDB) getOutputTotal(outid transaction.OutputID) (int64, error) {
	tx, ok := u.getTransaction(outid.TransactionID())
	if !ok {
		return 0, errors.New("no such transaction")
	}
	btmp, ok := tx.Outputs().Get(outid.Address())
	if !ok {
		return 0, errors.New("no such output")
	}
	bals := btmp.([]*balance.Balance)
	sum := int64(0)
	for _, b := range bals {
		sum += b.Value
	}
	return sum, nil
}

func (u *UtxoDB) checkLedgerBalance() {
	total := int64(0)
	for outp := range u.utxo {
		b, err := u.getOutputTotal(outp)
		if err != nil {
			panic("Wrong ledger balance: " + err.Error())
		}
		total += b
	}
	if total != supply {
		panic("wrong ledger balance")
	}
}

type AddressStats struct {
	Total      int64
	NumOutputs int
}

func (u *UtxoDB) GetLedgerStats() map[address.Address]AddressStats {
	u.mutex.RLock()
	defer u.mutex.RUnlock()

	ret := make(map[address.Address]AddressStats)
	for addr := range u.utxoByAddress {
		outputs := u.getAddressOutputs(addr)
		total := int64(0)
		for outp := range outputs {
			s, err := u.getOutputTotal(outp)
			if err != nil {
				panic(err)
			}
			total += s
		}
		ret[addr] = AddressStats{
			Total:      total,
			NumOutputs: len(outputs),
		}
	}
	return ret
}
