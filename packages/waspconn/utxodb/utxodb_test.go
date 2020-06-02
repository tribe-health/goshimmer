package utxodb

import (
	"github.com/iotaledger/goshimmer/dapps/valuetransfers/packages/address"
	"github.com/magiconair/properties/assert"
	"testing"
)

func TestBasic(t *testing.T) {
	genTx, ok := GetTransaction(genesisTxId)
	assert.Equal(t, ok, true)
	assert.Equal(t, genTx.ID(), genesisTxId)
}

func TestKnowAddresses(t *testing.T) {
	for i := 0; i < 4; i++ {
		sigS := GetSigScheme(GetAddress(i))
		t.Logf("#%d address: %s", i, sigS.Address().String())
	}
}

func TestGenesis(t *testing.T) {
	gout := GetAddressOutputs(GetGenesisSigScheme().Address())
	total := int64(0)
	for oid := range gout {
		sum, err := getOutputTotal(oid)
		assert.Equal(t, err, nil)
		total += sum
	}
	assert.Equal(t, total, supply-(int64(len(knownAddresses))-1)*ownerAmount)
	checkLedgerBalance()
}

func TestTransfer(t *testing.T) {
	_, err := DistributeIotas(1000000, GetGenesisAddress(), []address.Address{GetAddress(1)})
	assert.Equal(t, err, nil)
}

//
//func TestTransferAndBook(t *testing.T) {
//	tx, err := DistributeIotas(1000000, GetGenesisAddress(), GetAddress(1))
//	assert.Equal(t, err, nil)
//
//	err = AddTransaction(tx)
//	assert.Equal(t, err, nil)
//
//	tx, err = DistributeIotas(1000000, GetGenesisAddress(), GetAddress(2))
//	assert.Equal(t, err, nil)
//
//	err = AddTransaction(tx)
//	assert.Equal(t, err, nil)
//
//	tx, err = DistributeIotas(1000000, GetGenesisAddress(), GetAddress(3))
//	assert.Equal(t, err, nil)
//
//	err = AddTransaction(tx)
//	assert.Equal(t, err, nil)
//
//	stats := GetLedgerStats()
//	for addr, st := range stats {
//		t.Logf("%s: balance %d, num outputs %d", addr.String(), st.Total, st.NumOutputs)
//	}
//}
