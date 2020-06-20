package waspconn

import (
	"github.com/iotaledger/goshimmer/dapps/valuetransfers/packages/address"
	"github.com/iotaledger/goshimmer/dapps/valuetransfers/packages/balance"
	"github.com/iotaledger/goshimmer/dapps/valuetransfers/packages/transaction"
	"github.com/iotaledger/goshimmer/packages/waspconn"
	"github.com/iotaledger/goshimmer/packages/waspconn/chopper"
	"github.com/iotaledger/goshimmer/packages/waspconn/utxodb"
	"github.com/iotaledger/hive.go/netutil/buffconn"
	"io"
)

func (wconn *WaspConnector) sendMsgToWasp(msg interface{ Write(io.Writer) error }) error {
	data, err := waspconn.EncodeMsg(msg)
	if err != nil {
		return err
	}
	choppedData, chopped := chopper.ChopData(data)
	if !chopped {
		if len(data) > buffconn.MaxMessageSize {
			panic("sendMsgToWasp: internal inconsistency 1")
		}
		_, err = wconn.bconn.Write(data)
		return err
	}

	wconn.log.Debugf("+++++++++++++ %d bytes long message was split into %d chunks", len(data), len(choppedData))

	// sending piece by piece wrapped in WaspMsgChunk
	for _, piece := range choppedData {
		dataToSend, err := waspconn.EncodeMsg(&waspconn.WaspMsgChunk{
			Data: piece,
		})
		if err != nil {
			return err
		}
		if len(dataToSend) > buffconn.MaxMessageSize {
			panic("sendMsgToWasp: internal inconsistency 2")
		}
		_, err = wconn.bconn.Write(dataToSend)
		if err != nil {
			return err
		}
	}
	return nil
}

func (wconn *WaspConnector) sendTransactionToWasp(vtx *transaction.Transaction) error {
	return wconn.sendMsgToWasp(&waspconn.WaspFromNodeTransactionMsg{vtx})
}

func (wconn *WaspConnector) sendAddressUpdateToWasp(address *address.Address, balances map[transaction.ID][]*balance.Balance, tx *transaction.Transaction) error {
	return wconn.sendMsgToWasp(&waspconn.WaspFromNodeAddressUpdateMsg{
		Address:  *address,
		Balances: balances,
		Tx:       tx,
	})
}

func (wconn *WaspConnector) sendAddressOutputsToWasp(address *address.Address, balances map[transaction.ID][]*balance.Balance) error {
	return wconn.sendMsgToWasp(&waspconn.WaspFromNodeAddressOutputsMsg{
		Address:  *address,
		Balances: balances,
	})
}

// query outputs database and collects transactions containing unprocessed requests
func (wconn *WaspConnector) pushBacklogToWasp(addr *address.Address) {
	outs := utxodb.GetAddressOutputs(*addr)
	if len(outs) == 0 {
		return
	}
	outputs := waspconn.OutputsToBalances(outs)
	allColors := make(map[transaction.ID]bool)
	for _, bals := range outputs {
		for _, b := range bals {
			col := b.Color()
			if col == balance.ColorIOTA {
				continue
			}
			if col == balance.ColorNew {
				panic("unexpected balance.ColorNew")
			}

			allColors[(transaction.ID)(b.Color())] = true
		}
	}
	for txid := range allColors {
		tx, ok := utxodb.GetTransaction(txid)
		if !ok {
			wconn.log.Errorf("inconsistency: can't find txid = %s", txid.String())
			continue
		}
		if err := wconn.sendAddressUpdateToWasp(addr, outputs, tx); err != nil {
			wconn.log.Debug(err)
		}
	}
}
