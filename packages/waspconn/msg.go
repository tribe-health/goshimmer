package waspconn

import (
	"bytes"
	"fmt"
	"github.com/iotaledger/goshimmer/dapps/valuetransfers/packages/address"
	"github.com/iotaledger/goshimmer/dapps/valuetransfers/packages/balance"
	"github.com/iotaledger/goshimmer/dapps/valuetransfers/packages/transaction"
	"io"
)

const (
	waspPing = iota
	waspMsgChunk

	// wasp -> node
	waspToNodeTransaction
	waspToNodeSubscribe
	waspToNodeGetTransaction
	waspToNodeGetOutputs
	waspToNodeSetId

	// node -> wasp
	waspFromNodeTransaction
	waspFromNodeAddressUpdate
	waspFromNodeAddressOutputs
)

// special messages for big Data packets chopped into pieces
type WaspMsgChunk struct {
	Data []byte
}

type WaspPingMsg struct {
	Id        uint32
	Timestamp int64
}

type WaspToNodeTransactionMsg struct {
	Tx *transaction.Transaction
}

type WaspToNodeSubscribeMsg struct {
	Addresses []address.Address
}

type WaspToNodeGetTransactionMsg struct {
	TxId *transaction.ID
}

type WaspToNodeGetOutputsMsg struct {
	Address address.Address
}

type WaspToNodeSetIdMsg struct {
	Waspid string
}

type WaspFromNodeTransactionMsg struct {
	Tx *transaction.Transaction
}

type WaspFromNodeAddressUpdateMsg struct {
	Address  address.Address
	Balances map[transaction.ID][]*balance.Balance
	Tx       *transaction.Transaction
}

type WaspFromNodeAddressOutputsMsg struct {
	Address  address.Address
	Balances map[transaction.ID][]*balance.Balance
}

func typeToCode(msg interface{ Write(writer io.Writer) error }) byte {
	switch msg.(type) {
	case *WaspPingMsg:
		return waspPing

	case *WaspMsgChunk:
		return waspMsgChunk

	case *WaspToNodeTransactionMsg:
		return waspToNodeTransaction

	case *WaspToNodeSubscribeMsg:
		return waspToNodeSubscribe

	case *WaspToNodeGetTransactionMsg:
		return waspToNodeGetTransaction

	case *WaspToNodeGetOutputsMsg:
		return waspToNodeGetOutputs

	case *WaspToNodeSetIdMsg:
		return waspToNodeSetId

	case *WaspFromNodeTransactionMsg:
		return waspFromNodeTransaction

	case *WaspFromNodeAddressUpdateMsg:
		return waspFromNodeAddressUpdate

	case *WaspFromNodeAddressOutputsMsg:
		return waspFromNodeAddressOutputs
	}
	panic("wrong type")
}

func EncodeMsg(msg interface{ Write(writer io.Writer) error }) ([]byte, error) {
	msgCode := typeToCode(msg)
	var buf bytes.Buffer

	if err := buf.WriteByte(msgCode); err != nil {
		return nil, err
	}
	if err := msg.Write(&buf); err != nil {
		return nil, err
	}
	ret := buf.Bytes()
	return ret, nil
}

func DecodeMsg(data []byte, waspSide bool) (interface{}, error) {
	if len(data) < 1 {
		return nil, fmt.Errorf("wrong message")
	}
	var ret interface{ Read(io.Reader) error }

	switch data[0] {
	case waspPing:
		ret = &WaspPingMsg{}

	case waspMsgChunk:
		ret = &WaspMsgChunk{}

	case waspToNodeTransaction:
		if waspSide {
			return nil, fmt.Errorf("wrong message")
		}
		ret = &WaspToNodeTransactionMsg{}

	case waspToNodeSubscribe:
		if waspSide {
			return nil, fmt.Errorf("wrong message")
		}
		ret = &WaspToNodeSubscribeMsg{}

	case waspToNodeGetTransaction:
		if waspSide {
			return nil, fmt.Errorf("wrong message")
		}
		ret = &WaspToNodeGetTransactionMsg{}

	case waspToNodeGetOutputs:
		if waspSide {
			return nil, fmt.Errorf("wrong message")
		}
		ret = &WaspToNodeGetOutputsMsg{}

	case waspToNodeSetId:
		if waspSide {
			return nil, fmt.Errorf("wrong message")
		}
		ret = &WaspToNodeSetIdMsg{}

	case waspFromNodeTransaction:
		if !waspSide {
			return nil, fmt.Errorf("wrong message")
		}
		ret = &WaspFromNodeTransactionMsg{}

	case waspFromNodeAddressUpdate:
		if !waspSide {
			return nil, fmt.Errorf("wrong message")
		}
		ret = &WaspFromNodeAddressUpdateMsg{}

	case waspFromNodeAddressOutputs:
		if !waspSide {
			return nil, fmt.Errorf("wrong message")
		}
		ret = &WaspFromNodeAddressOutputsMsg{}

	default:
		return nil, fmt.Errorf("wrong message code")
	}
	if err := ret.Read(bytes.NewReader(data[1:])); err != nil {
		return nil, err
	}
	return ret, nil
}

func (msg *WaspPingMsg) Write(w io.Writer) error {
	if err := WriteUint32(w, msg.Id); err != nil {
		return err
	}
	if err := WriteUint64(w, uint64(msg.Timestamp)); err != nil {
		return err
	}
	return nil
}

func (msg *WaspPingMsg) Read(r io.Reader) error {
	if err := ReadUint32(r, &msg.Id); err != nil {
		return err
	}
	var ts uint64
	if err := ReadUint64(r, &ts); err != nil {
		return err
	}
	msg.Timestamp = int64(ts)
	return nil
}

func (msg *WaspToNodeTransactionMsg) Write(w io.Writer) error {
	return WriteBytes32(w, msg.Tx.Bytes())
}

func (msg *WaspToNodeTransactionMsg) Read(r io.Reader) error {
	var err error
	data, err := ReadBytes32(r)
	if err != nil {
		return err
	}
	msg.Tx, _, err = transaction.FromBytes(data)
	return err
}

func (msg *WaspToNodeSubscribeMsg) Write(w io.Writer) error {
	if err := WriteUint16(w, uint16(len(msg.Addresses))); err != nil {
		return err
	}
	for _, addr := range msg.Addresses {
		if _, err := w.Write(addr[:]); err != nil {
			return err
		}
	}
	return nil
}

func (msg *WaspToNodeSubscribeMsg) Read(r io.Reader) error {
	var size uint16
	if err := ReadUint16(r, &size); err != nil {
		return err
	}
	msg.Addresses = make([]address.Address, size)
	for i := range msg.Addresses {
		if err := ReadAddress(r, &msg.Addresses[i]); err != nil {
			return err
		}
	}
	return nil
}

func (msg *WaspToNodeGetTransactionMsg) Write(w io.Writer) error {
	_, err := w.Write(msg.TxId.Bytes())
	return err
}

func (msg *WaspToNodeGetTransactionMsg) Read(r io.Reader) error {
	msg.TxId = new(transaction.ID)
	n, err := r.Read(msg.TxId[:])
	if err != nil {
		return err
	}
	if n != transaction.IDLength {
		return fmt.Errorf("error while reading 'get transaction' message")
	}
	return nil
}

func (msg *WaspToNodeGetOutputsMsg) Write(w io.Writer) error {
	_, err := w.Write(msg.Address[:])
	return err
}

func (msg *WaspToNodeGetOutputsMsg) Read(r io.Reader) error {
	return ReadAddress(r, &msg.Address)
}

func (msg *WaspToNodeSetIdMsg) Write(w io.Writer) error {
	return WriteString(w, msg.Waspid)
}

func (msg *WaspToNodeSetIdMsg) Read(r io.Reader) error {
	var err error
	msg.Waspid, err = ReadString(r)
	return err
}

func (msg *WaspFromNodeTransactionMsg) Write(w io.Writer) error {
	return WriteBytes32(w, msg.Tx.Bytes())
}

func (msg *WaspFromNodeTransactionMsg) Read(r io.Reader) error {
	data, err := ReadBytes32(r)
	if err != nil {
		return err
	}
	msg.Tx, _, err = transaction.FromBytes(data)
	return err
}

func (msg *WaspFromNodeAddressUpdateMsg) Write(w io.Writer) error {
	_, err := w.Write(msg.Address[:])
	if err != nil {
		return err
	}
	if err := WriteBalances(w, msg.Balances); err != nil {
		return err
	}
	return WriteBytes32(w, msg.Tx.Bytes())
}

func (msg *WaspFromNodeAddressUpdateMsg) Read(r io.Reader) error {
	var err error
	if err = ReadAddress(r, &msg.Address); err != nil {
		return err
	}
	if msg.Balances, err = ReadBalances(r); err != nil {
		return err
	}
	data, err := ReadBytes32(r)
	if err != nil {
		return err
	}
	msg.Tx, _, err = transaction.FromBytes(data)
	return err
}

func (msg *WaspFromNodeAddressOutputsMsg) Write(w io.Writer) error {
	_, err := w.Write(msg.Address[:])
	if err != nil {
		return err
	}
	return WriteBalances(w, msg.Balances)
}

func (msg *WaspMsgChunk) Read(r io.Reader) error {
	var err error
	msg.Data, err = ReadBytes16(r)
	return err
}

func (msg *WaspMsgChunk) Write(w io.Writer) error {
	return WriteBytes16(w, msg.Data)
}

func (msg *WaspFromNodeAddressOutputsMsg) Read(r io.Reader) error {
	var err error
	if err = ReadAddress(r, &msg.Address); err != nil {
		return err
	}
	msg.Balances, err = ReadBalances(r)
	return err
}

func WriteBalances(w io.Writer, balances map[transaction.ID][]*balance.Balance) error {
	if err := ValidateBalances(balances); err != nil {
		return err
	}
	if err := WriteUint16(w, uint16(len(balances))); err != nil {
		return err
	}
	for txid, bals := range balances {
		if _, err := w.Write(txid[:]); err != nil {
			return err
		}
		if err := WriteUint16(w, uint16(len(bals))); err != nil {
			return err
		}
		for _, b := range bals {
			if _, err := w.Write(b.Color().Bytes()); err != nil {
				return err
			}
			if err := WriteUint64(w, uint64(b.Value())); err != nil {
				return err
			}
		}
	}
	return nil
}

func ReadBalances(r io.Reader) (map[transaction.ID][]*balance.Balance, error) {
	var size uint16
	if err := ReadUint16(r, &size); err != nil {
		return nil, err
	}
	ret := make(map[transaction.ID][]*balance.Balance, size)
	for i := uint16(0); i < size; i++ {
		var txid transaction.ID
		if err := ReadTransactionId(r, &txid); err != nil {
			return nil, err
		}
		var numBals uint16
		if err := ReadUint16(r, &numBals); err != nil {
			return nil, err
		}
		lst := make([]*balance.Balance, numBals)
		for i := range lst {
			var color balance.Color
			if err := ReadColor(r, &color); err != nil {
				return nil, err
			}
			var value uint64
			if err := ReadUint64(r, &value); err != nil {
				return nil, err
			}
			lst[i] = balance.New(color, int64(value))
		}
		ret[txid] = lst
	}
	if err := ValidateBalances(ret); err != nil {
		return nil, err
	}
	return ret, nil
}

func OutputsToBalances(outs map[transaction.OutputID][]*balance.Balance) map[transaction.ID][]*balance.Balance {
	ret := make(map[transaction.ID][]*balance.Balance)
	var niltxid transaction.ID

	for outp, bals := range outs {
		if outp.TransactionID() == niltxid {
			panic("outp.TransactionID() == niltxid")
		}
		ret[outp.TransactionID()] = bals
	}
	return ret
}

func BalancesToOutputs(addr address.Address, bals map[transaction.ID][]*balance.Balance) map[transaction.OutputID][]*balance.Balance {
	ret := make(map[transaction.OutputID][]*balance.Balance)
	var niltxid transaction.ID

	for txid, bal := range bals {
		if txid == niltxid {
			panic("txid == niltxid")
		}
		ret[transaction.NewOutputID(addr, txid)] = bal
	}
	return ret
}
