package utxodb

import (
	"fmt"
	"github.com/iotaledger/goshimmer/dapps/valuetransfers/packages/transaction"
	"math/rand"
	"sync"
	"time"
)

type pendingTransaction struct {
	confirmDeadline time.Time
	tx              *transaction.Transaction
	hasConflicts    bool
	onConfirm       func()
}

type confirmEmulator struct {
	confirmTime            time.Duration
	randomize              bool
	confirmFirstInConflict bool
	pendingTransactions    map[transaction.ID]*pendingTransaction
}

var (
	Confirm      confirmEmulator
	confirmMutex sync.Mutex
)

func init() {
	Confirm = confirmEmulator{
		pendingTransactions: make(map[transaction.ID]*pendingTransaction),
	}
	go confirmLoop()
}

func SetConfirmationParams(confTime time.Duration, randomize, confirmFirstInConflict bool) {
	confirmMutex.Lock()
	defer confirmMutex.Unlock()
	Confirm.confirmTime = confTime
	Confirm.randomize = randomize
	Confirm.confirmFirstInConflict = confirmFirstInConflict
}

func (c *confirmEmulator) AddTransaction(tx *transaction.Transaction, onConfirm func()) error {
	if onConfirm == nil {
		onConfirm = func() {}
	}
	confirmMutex.Lock()
	defer confirmMutex.Unlock()

	if Confirm.confirmTime == 0 {
		if err := AddTransaction(tx); err != nil {
			return err
		}
		onConfirm()
		fmt.Printf("utxodb.ConfirmEmulator CONFIRMED IMMEDIATELY: %s\n", tx.ID().String())
		return nil
	}
	if err := ValidateTransaction(tx); err != nil {
		return err
	}
	for txid, ptx := range Confirm.pendingTransactions {
		if AreConflicting(tx, ptx.tx) {
			ptx.hasConflicts = true
			return fmt.Errorf("utxodb.ConfirmEmulator rejected: new tx %s conflicts with pending tx %s", tx.ID().String(), txid.String())
		}
	}
	var confTime time.Duration
	if Confirm.randomize {
		confTime = time.Duration(rand.Int31n(int32(Confirm.confirmTime)) + int32(Confirm.confirmTime)/2)
	} else {
		confTime = Confirm.confirmTime
	}
	deadline := time.Now().Add(confTime)

	Confirm.pendingTransactions[tx.ID()] = &pendingTransaction{
		confirmDeadline: deadline,
		tx:              tx,
		hasConflicts:    false,
		onConfirm:       onConfirm,
	}
	fmt.Printf("utxodb.ConfirmEmulator ADDED PENDING TRANSACTION: %s\n", tx.ID().String())
	return nil
}

const loopPeriod = 500 * time.Millisecond

func confirmLoop() {
	maturedTxs := make([]transaction.ID, 0)
	for {
		time.Sleep(loopPeriod)

		maturedTxs = maturedTxs[:0]
		nowis := time.Now()
		confirmMutex.Lock()

		for txid, ptx := range Confirm.pendingTransactions {
			if ptx.confirmDeadline.Before(nowis) {
				maturedTxs = append(maturedTxs, txid)
			}
		}

		if len(maturedTxs) == 0 {
			confirmMutex.Unlock()
			continue
		}

		for _, txid := range maturedTxs {
			ptx := Confirm.pendingTransactions[txid]
			if ptx.hasConflicts && !Confirm.confirmFirstInConflict {
				// do not confirm if tx has conflicts
				fmt.Printf("!!! utxodb.ConfirmEmulator: rejected because has conflicts %s\n", txid.String())
				continue
			}
			if err := AddTransaction(ptx.tx); err != nil {
				fmt.Printf("!!!! utxodb.AddTransaction: %v\n", err)
			} else {
				ptx.onConfirm()
				fmt.Printf("+++ utxodb.ConfirmEmulator: CONFIRMED %s after %v\n", txid.String(), Confirm.confirmTime)
			}
		}

		for _, txid := range maturedTxs {
			delete(Confirm.pendingTransactions, txid)
		}
		confirmMutex.Unlock()
	}
}
