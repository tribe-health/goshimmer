package waspconn

import (
	"bytes"
	"github.com/iotaledger/goshimmer/packages/waspconn/utxodb"
	"github.com/magiconair/properties/assert"
	"testing"
)

func TestBalances(t *testing.T) {
	addr := utxodb.GetAddress(2)
	outs := utxodb.GetAddressOutputs(addr)
	var buf bytes.Buffer
	bals := OutputsToBalances(outs)

	err := WriteBalances(&buf, bals)
	assert.Equal(t, err, nil)

	balsBack, err := ReadBalances(bytes.NewReader(buf.Bytes()))
	assert.Equal(t, err, nil)

	var bufBack bytes.Buffer
	err = WriteBalances(&bufBack, balsBack)

	assert.Equal(t, bytes.Equal(buf.Bytes(), bufBack.Bytes()), true)

	assert.Equal(t, err, nil)

	_ = BalancesToOutputs(addr, balsBack)
}
