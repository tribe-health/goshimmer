package utxodb

import (
	"github.com/iotaledger/goshimmer/dapps/valuetransfers/packages/address/signaturescheme"
	"github.com/iotaledger/hive.go/crypto/ed25519"
	"testing"
)

func TestGenOwnerAddress(t *testing.T) {
	for i := 4; i <= 10; i++ {
		keyPair := ed25519.GenerateKeyPair()
		t.Logf("ownerPrivKey%d = \"%s\"", i, keyPair.PrivateKey.String())
		t.Logf("ownerPubKey%d = \"%s\"", i, keyPair.PublicKey.String())
		sigscheme := signaturescheme.ED25519(keyPair)
		t.Logf("ownerAddress%d = \"%s\"", i, sigscheme.Address().String())
	}
}
