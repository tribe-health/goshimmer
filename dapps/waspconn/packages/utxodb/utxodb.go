package utxodb

import (
	"errors"
	"fmt"
	"github.com/iotaledger/goshimmer/dapps/valuetransfers/packages/address"
	"github.com/iotaledger/goshimmer/dapps/valuetransfers/packages/balance"
	"github.com/iotaledger/goshimmer/dapps/valuetransfers/packages/transaction"
	"sync"
)

var (
	transactions  map[transaction.ID]*transaction.Transaction
	utxo          map[transaction.OutputID]bool
	utxoByAddress map[address.Address][]transaction.ID
	mutexdb       *sync.RWMutex
)

func init() {
	Init()
	fmt.Printf("UTXODB initialized\n")
}

// PrintPredefined prints basic value and stats of predefined addresses
func PrintPredefined() {
	stats := GetLedgerStats()
	fmt.Printf("UTXODB initialized:\nSeed: %s\nTotal supply = %di\nGenesis + %d predefined addresses with %di each\n",
		seedStr, supply, len(sigSchemes)-1, ownerAmount)

	fmt.Println("Balances:")
	for i, sigScheme := range sigSchemes {
		addr := sigScheme.Address()
		fmt.Printf("#%d: %s: balance %d, num outputs %d\n", i, addr.String(), stats[addr].Total, stats[addr].NumOutputs)
	}
}

func Init() {
	transactions = make(map[transaction.ID]*transaction.Transaction)
	utxo = make(map[transaction.OutputID]bool)
	utxoByAddress = make(map[address.Address][]transaction.ID)
	mutexdb = &sync.RWMutex{}

	genesisInit()
}

func ValidateTransaction(tx *transaction.Transaction) error {
	if err := CheckInputsOutputs(tx); err != nil {
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

func IsConfirmed(txid *transaction.ID) bool {
	mutexdb.Lock()
	defer mutexdb.Unlock()
	_, ok := transactions[*txid]
	return ok
}

func AddTransaction(tx *transaction.Transaction) error {
	//fmt.Printf("[utxodb] AddTransaction::\n%s\n", tx.String())
	if err := ValidateTransaction(tx); err != nil {
		return err
	}

	mutexdb.Lock()
	defer mutexdb.Unlock()

	if _, ok := transactions[tx.ID()]; ok {
		return fmt.Errorf("duplicate transaction txid = %s", tx.ID().String())
	}

	var err error

	// check if outputs exist
	tx.Inputs().ForEach(func(outputId transaction.OutputID) bool {
		if _, ok := utxo[outputId]; !ok {
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
		delete(utxo, outputId)
		lst, ok := utxoByAddress[outputId.Address()]
		if ok {
			newLst := make([]transaction.ID, 0, len(lst))
			for _, txid := range lst {
				if txid != outputId.TransactionID() {
					newLst = append(newLst, txid)
				}
			}
			utxoByAddress[outputId.Address()] = newLst
		}
		return true
	})

	tx.Outputs().ForEach(func(addr address.Address, bals []*balance.Balance) bool {
		utxo[transaction.NewOutputID(addr, tx.ID())] = true
		lst, ok := utxoByAddress[addr]
		if !ok {
			lst = make([]transaction.ID, 0)
		}
		lst = append(lst, tx.ID())
		utxoByAddress[addr] = lst
		return true
	})
	transactions[tx.ID()] = tx
	checkLedgerBalance()
	return nil
}

func GetTransaction(id transaction.ID) (*transaction.Transaction, bool) {
	mutexdb.RLock()
	defer mutexdb.RUnlock()

	return getTransaction(id)
}

func getTransaction(id transaction.ID) (*transaction.Transaction, bool) {
	tx, ok := transactions[id]
	return tx, ok
}

func mustGetTransaction(id transaction.ID) *transaction.Transaction {
	tx, ok := transactions[id]
	if !ok {
		panic(fmt.Sprintf("tx id doesn't exist: %s", id.String()))
	}
	return tx
}

func MustGetTransaction(id transaction.ID) *transaction.Transaction {
	mutexdb.RLock()
	defer mutexdb.RUnlock()
	return mustGetTransaction(id)
}

func GetAddressOutputs(addr address.Address) map[transaction.OutputID][]*balance.Balance {
	mutexdb.RLock()
	defer mutexdb.RUnlock()

	return getAddressOutputs(addr)
}

func getAddressOutputs(addr address.Address) map[transaction.OutputID][]*balance.Balance {
	ret := make(map[transaction.OutputID][]*balance.Balance)

	txIds, ok := utxoByAddress[addr]
	if !ok || len(txIds) == 0 {
		return nil
	}
	var nilid transaction.ID
	for _, txid := range txIds {
		if txid == nilid {
			panic("txid == nilid")
		}
		txInp := mustGetTransaction(txid)
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

func getOutputTotal(outid transaction.OutputID) (int64, error) {
	tx, ok := getTransaction(outid.TransactionID())
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

func checkLedgerBalance() {
	total := int64(0)
	for outp := range utxo {
		b, err := getOutputTotal(outp)
		if err != nil {
			panic("Wrong ledger balance: " + err.Error())
		}
		total += b
	}
	if total != GetSupply() {
		panic("wrong ledger balance")
	}
}

type AddressStats struct {
	Total      int64
	NumOutputs int
}

func GetLedgerStats() map[address.Address]AddressStats {
	mutexdb.RLock()
	defer mutexdb.RUnlock()

	ret := make(map[address.Address]AddressStats)
	for addr := range utxoByAddress {
		outputs := getAddressOutputs(addr)
		total := int64(0)
		for outp := range outputs {
			s, err := getOutputTotal(outp)
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
