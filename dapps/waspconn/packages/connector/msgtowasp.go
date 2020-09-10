package connector

import (
	"github.com/iotaledger/goshimmer/dapps/valuetransfers/packages/address"
	"github.com/iotaledger/goshimmer/dapps/valuetransfers/packages/balance"
	"github.com/iotaledger/goshimmer/dapps/valuetransfers/packages/transaction"
	"github.com/iotaledger/goshimmer/dapps/waspconn/packages/chopper"
	"github.com/iotaledger/goshimmer/dapps/waspconn/packages/waspconn"
	"github.com/iotaledger/goshimmer/packages/binary/messagelayer/payload"
	"io"
)

func (wconn *WaspConnector) sendMsgToWasp(msg interface{ Write(io.Writer) error }) error {
	data, err := waspconn.EncodeMsg(msg)
	if err != nil {
		return err
	}
	choppedData, chopped := chopper.ChopData(data, payload.MaxMessageSize-waspconn.ChunkMessageHeaderSize)
	if !chopped {
		if len(data) > payload.MaxMessageSize {
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
		if len(dataToSend) > payload.MaxMessageSize {
			wconn.log.Panicf("sendMsgToWasp: internal inconsistency 3 size too big: %d", len(dataToSend))
		}
		_, err = wconn.bconn.Write(dataToSend)
		if err != nil {
			return err
		}
	}
	return nil
}

func (wconn *WaspConnector) sendConfirmedTransactionToWasp(vtx *transaction.Transaction) error {
	return wconn.sendMsgToWasp(&waspconn.WaspFromNodeConfirmedTransactionMsg{
		Tx: vtx,
	})
}

func (wconn *WaspConnector) sendAddressUpdateToWasp(addr *address.Address, balances map[transaction.ID][]*balance.Balance, tx *transaction.Transaction) error {
	return wconn.sendMsgToWasp(&waspconn.WaspFromNodeAddressUpdateMsg{
		Address:  *addr,
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

func (wconn *WaspConnector) sendTxInclusionLevelToWasp(inclLevel byte, txid *transaction.ID, addrs []address.Address) error {
	return wconn.sendMsgToWasp(&waspconn.WaspFromNodeTransactionInclusionLevelMsg{
		Level:               inclLevel,
		TxId:                *txid,
		SubscribedAddresses: addrs,
	})
}

// query outputs database and collects transactions containing unprocessed requests
func (wconn *WaspConnector) pushBacklogToWasp(addr *address.Address) {
	wconn.log.Infow("pushBacklogToWasp", "addr", addr.String())

	outs, err := wconn.vtangle.GetConfirmedAddressOutputs(*addr)
	if err != nil {
		wconn.log.Error(err)
		return
	}
	if len(outs) == 0 {
		return
	}
	balances := waspconn.OutputsToBalances(outs)
	wconn.log.Infof("pushBacklogToWasp. addr: %s, balances:\n%s\n",
		addr.String(), waspconn.BalancesToString(balances))

	// collect all colors of balances
	allColorsMap := make(map[transaction.ID]bool)
	for _, bals := range balances {
		for _, b := range bals {
			col := b.Color
			if col == balance.ColorIOTA {
				continue
			}
			if col == balance.ColorNew {
				wconn.log.Errorf("unexpected balance.ColorNew")
				continue
			}
			allColorsMap[(transaction.ID)(b.Color)] = true
		}
	}
	allColors := make([]transaction.ID, 0, len(allColorsMap))
	for addr := range allColorsMap {
		allColors = append(allColors, addr)
	}
	wconn.log.Infof("pushBacklogToWasp: allColors: %+v\n", allColors)

	// for each color we try to load corresponding origin transaction.
	// if the transaction exist and it is among the balances of the address,
	// the send balances with the transaction as address update
	sentTxs := make([]transaction.ID, 0)
	for _, txid := range allColors {
		tx := wconn.vtangle.GetConfirmedTransaction(&txid)
		if tx == nil {
			wconn.log.Warnf("can't find the origin tx for the color %s. It may be snapshotted", txid.String())
			continue
		}
		if _, ok := balances[txid]; !ok {
			// the transaction taken by color is not among transaction in balances of the address.
			// Irrelevant to the backlog, skip it.
			continue
		}

		wconn.log.Infof("pushBacklogToWasp: sending update with tx: %s\n", tx.String())

		if err := wconn.sendAddressUpdateToWasp(addr, balances, tx); err != nil {
			wconn.log.Errorf("sendAddressUpdateToWasp: %v", err)
		} else {
			sentTxs = append(sentTxs, txid)
		}
	}
	wconn.log.Infof("backlog -> Wasp for addr: %s, txs: %+v", addr.String(), sentTxs)
}
