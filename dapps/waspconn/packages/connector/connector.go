package connector

import (
	"io"
	"net"
	"strings"

	"github.com/iotaledger/goshimmer/dapps/valuetransfers/packages/address"
	"github.com/iotaledger/goshimmer/dapps/valuetransfers/packages/balance"
	"github.com/iotaledger/goshimmer/dapps/valuetransfers/packages/transaction"
	"github.com/iotaledger/goshimmer/dapps/waspconn/packages/valuetangle"
	"github.com/iotaledger/goshimmer/dapps/waspconn/packages/waspconn"
	"github.com/iotaledger/goshimmer/packages/binary/messagelayer/payload"
	"github.com/iotaledger/goshimmer/packages/shutdown"
	"github.com/iotaledger/hive.go/daemon"
	"github.com/iotaledger/hive.go/events"
	"github.com/iotaledger/hive.go/logger"
	"github.com/iotaledger/hive.go/netutil/buffconn"
)

type WaspConnector struct {
	id                                      string
	bconn                                   *buffconn.BufferedConnection
	subscriptions                           map[address.Address]int
	inTxChan                                chan *transaction.Transaction
	exitConnChan                            chan struct{}
	receiveConfirmedValueTransactionClosure *events.Closure
	receiveWaspMessageClosure               *events.Closure
	closeClosure                            *events.Closure
	log                                     *logger.Logger
	vtangle                                 valuetangle.ValueTangle
}

func Run(conn net.Conn, log *logger.Logger, vtangle valuetangle.ValueTangle) {
	wconn := &WaspConnector{
		bconn:        buffconn.NewBufferedConnection(conn, payload.MaxMessageSize),
		exitConnChan: make(chan struct{}),
		log:          log,
		vtangle:      vtangle,
	}
	err := daemon.BackgroundWorker(wconn.Id(), func(shutdownSignal <-chan struct{}) {
		select {
		case <-shutdownSignal:
			wconn.log.Infof("shutdown signal received")
			_ = wconn.bconn.Close()

		case <-wconn.exitConnChan:
			wconn.log.Infof("closing..")
			_ = wconn.bconn.Close()
		}

		go wconn.detach()
	}, shutdown.PriorityWaspConn)

	if err != nil {
		close(wconn.exitConnChan)
		wconn.log.Errorf("can't start a deamon")
		return
	}
	wconn.attach()
}

func (wconn *WaspConnector) Id() string {
	if wconn.id == "" {
		return "wasp_" + wconn.bconn.RemoteAddr().String()
	}
	return wconn.id
}

func (wconn *WaspConnector) SetId(id string) {
	wconn.id = id
	wconn.log = wconn.log.Named(id)
	wconn.log.Infof("wasp connection id has been set to '%s' for '%s'", id, wconn.bconn.RemoteAddr().String())
}

func (wconn *WaspConnector) attach() {
	wconn.subscriptions = make(map[address.Address]int)
	wconn.inTxChan = make(chan *transaction.Transaction)

	wconn.receiveConfirmedValueTransactionClosure = events.NewClosure(func(vtx *transaction.Transaction) {
		wconn.inTxChan <- vtx
	})

	wconn.receiveWaspMessageClosure = events.NewClosure(func(data []byte) {
		wconn.processMsgDataFromWasp(data)
	})

	wconn.closeClosure = events.NewClosure(func() {
		wconn.log.Info("Wasp connection closed")
	})

	// attach connector to the flow of incoming value transactions
	EventValueTransactionConfirmed.Attach(wconn.receiveConfirmedValueTransactionClosure)

	wconn.bconn.Events.ReceiveMessage.Attach(wconn.receiveWaspMessageClosure)
	wconn.bconn.Events.Close.Attach(wconn.closeClosure)

	// read connection thread
	go func() {
		if err := wconn.bconn.Read(); err != nil {
			if err != io.EOF && !strings.Contains(err.Error(), "use of closed network connection") {
				wconn.log.Warnw("Permanent error", "err", err)
			}
		}
		close(wconn.exitConnChan)
	}()

	// read incoming pre-filtered transactions from node
	go func() {
		for vtx := range wconn.inTxChan {
			wconn.processConfirmedTransactionFromNode(vtx)
		}
	}()
}

func (wconn *WaspConnector) detach() {
	wconn.vtangle.Detach()
	EventValueTransactionConfirmed.Detach(wconn.receiveConfirmedValueTransactionClosure)
	wconn.bconn.Events.ReceiveMessage.Detach(wconn.receiveWaspMessageClosure)
	wconn.bconn.Events.Close.Detach(wconn.closeClosure)

	close(wconn.inTxChan)
	_ = wconn.bconn.Close()

	wconn.log.Debugf("detached waspconn")
}

func (wconn *WaspConnector) subscribe(addr *address.Address) {
	_, ok := wconn.subscriptions[*addr]
	if !ok {
		wconn.log.Debugf("subscribed to address: %s", addr.String())
		wconn.subscriptions[*addr] = 0
	}
}

func (wconn *WaspConnector) isSubscribed(addr *address.Address) bool {
	_, ok := wconn.subscriptions[*addr]
	return ok
}

// processConfirmedTransactionFromNode receives only confirmed transactions
// it parses SC transaction incoming from the node. Forwards it to Wasp if subscribed
func (wconn *WaspConnector) processConfirmedTransactionFromNode(tx *transaction.Transaction) {
	// determine if transaction contains any of subscribed addresses in its outputs
	wconn.log.Debugw("processConfirmedTransactionFromNode", "txid", tx.ID().String())

	subscribedOutAddresses := make([]address.Address, 0)
	tx.Outputs().ForEach(func(addr address.Address, _ []*balance.Balance) bool {
		if wconn.isSubscribed(&addr) {
			subscribedOutAddresses = append(subscribedOutAddresses, addr)
		}
		return true
	})
	if len(subscribedOutAddresses) == 0 {
		wconn.log.Debugw("not subscribed", "txid", tx.ID().String())
		// dismiss unsubscribed transaction
		return
	}
	// for each subscribed address retrieve outputs and send to wasp with the transaction
	wconn.log.Debugf("txid %s contains %d subscribed addresses", tx.ID().String(), len(subscribedOutAddresses))

	for i := range subscribedOutAddresses {
		outs, err := wconn.vtangle.GetConfirmedAddressOutputs(subscribedOutAddresses[i])
		if err != nil {
			wconn.log.Error(err)
			continue
		}
		// for each subscribed address send its confirmed UTXOs to Wasp as separate message
		err = wconn.sendAddressUpdateToWasp(
			&subscribedOutAddresses[i],
			waspconn.OutputsToBalances(outs),
			tx,
		)
		if err != nil {
			wconn.log.Error(err)
		}
	}
}

// find transaction async, parse it to SCTransaction and send to Wasp
func (wconn *WaspConnector) getTransaction(txid *transaction.ID) {
	wconn.log.Debugf("requested transaction id = %s", txid.String())

	tx, confirmed := wconn.vtangle.GetTransaction(txid)
	if tx == nil {
		wconn.log.Debugf("!!!! GetTransaction %s : not found", txid.String())
		return
	}
	if err := wconn.sendTransactionToWasp(tx, confirmed); err != nil {
		wconn.log.Debugf("!!!! sendTransactionToWasp: %v", err)
		return
	}
}

func (wconn *WaspConnector) getAddressBalance(addr *address.Address) {
	wconn.log.Debugf("getAddressBalance request for address: %s", addr.String())

	outputs, err := wconn.vtangle.GetConfirmedAddressOutputs(*addr)
	if err != nil {
		panic(err)
	}
	if len(outputs) == 0 {
		return
	}
	ret := waspconn.OutputsToBalances(outputs)

	wconn.log.Debugf("sending balances to wasp: %s    %+v", addr.String(), ret)

	if err := wconn.sendAddressOutputsToWasp(addr, ret); err != nil {
		wconn.log.Debugf("sendAddressOutputsToWasp: %v", err)
	}
}

func (wconn *WaspConnector) postTransaction(tx *transaction.Transaction) {
	if err := wconn.vtangle.PostTransaction(tx); err != nil {
		wconn.log.Warn(err)
		return
	}
	wconn.log.Debugf("Posted transaction %s", tx.ID().String())
}
