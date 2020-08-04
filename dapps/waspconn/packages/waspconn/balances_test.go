package waspconn

import (
	"bytes"
	"github.com/iotaledger/goshimmer/dapps/waspconn/packages/utxodb"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestBalances(t *testing.T) {
	u := utxodb.New()

	addr := utxodb.NewSigScheme("C6hPhCS2E2dKUGS3qj4264itKXohwgL3Lm2fNxayAKr", 0).Address()
	_, err := u.RequestFunds(addr)
	assert.NoError(t, err)

	outs := u.GetAddressOutputs(addr)
	var buf bytes.Buffer
	bals := OutputsToBalances(outs)

	err = WriteBalances(&buf, bals)
	assert.Equal(t, err, nil)

	balsBack, err := ReadBalances(bytes.NewReader(buf.Bytes()))
	assert.Equal(t, err, nil)

	var bufBack bytes.Buffer
	err = WriteBalances(&bufBack, balsBack)

	assert.Equal(t, bytes.Equal(buf.Bytes(), bufBack.Bytes()), true)

	assert.Equal(t, err, nil)

	_ = BalancesToOutputs(addr, balsBack)
}
