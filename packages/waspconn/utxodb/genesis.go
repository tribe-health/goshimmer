package utxodb

import (
	"fmt"
	"github.com/iotaledger/goshimmer/dapps/valuetransfers/packages/address"
	"github.com/iotaledger/goshimmer/dapps/valuetransfers/packages/address/signaturescheme"
	"github.com/iotaledger/goshimmer/dapps/valuetransfers/packages/balance"
	"github.com/iotaledger/goshimmer/dapps/valuetransfers/packages/transaction"
	"github.com/iotaledger/hive.go/crypto/ed25519"
	"github.com/mr-tron/base58"
)

const (
	supply            = int64(100 * 1000 * 1000 * 1000)
	ownerAmount       = 1000 * 1000 * 1000
	seedStr           = "EFonzaUz5ngYeDxbRKu8qV5aoSogUQ5qVSTSjn7hJ8FQ"
	numKnownAddresses = 11 // including genesis
)

var (
	genesisTxId transaction.ID

	sigSchemes          = make([]signaturescheme.SignatureScheme, numKnownAddresses)
	sigSchemesByAddress = make(map[address.Address]signaturescheme.SignatureScheme)
)

func init() {
	seedBin, err := base58.Decode(seedStr)
	if err != nil {
		panic(err)
	}
	seed := ed25519.NewSeed(seedBin)

	// generate range of signature schemes and addresses form the seed
	// index 0 i considered to be the origin
	for i := range sigSchemes {
		keyPair := seed.KeyPair(uint64(i))
		sigSchemes[i] = signaturescheme.ED25519(*keyPair)
		sigSchemesByAddress[sigSchemes[i].Address()] = sigSchemes[i]
	}
	// create genesis transaction

	genesisInput := transaction.NewOutputID(GetGenesisAddress(), transaction.ID{})
	inputs := transaction.NewInputs(genesisInput)
	outputs := transaction.NewOutputs(map[address.Address][]*balance.Balance{
		GetGenesisAddress(): {balance.New(balance.ColorIOTA, supply)},
	})
	genesisTx := transaction.New(inputs, outputs)
	genesisTx.Sign(GetGenesisSigScheme())

	genesisTxId = genesisTx.ID()

	transactions[genesisTxId] = genesisTx
	utxo[transaction.NewOutputID(GetGenesisAddress(), genesisTxId)] = true
	utxoByAddress[GetGenesisAddress()] = []transaction.ID{genesisTxId}

	testAddresses := make([]address.Address, len(sigSchemes)-1)
	for i := range testAddresses {
		testAddresses[i] = GetAddress(i + 1)
	}

	tx, err := DistributeIotas(ownerAmount, GetGenesisAddress(), testAddresses)
	if err != nil {
		panic(err)
	}
	if err = AddTransaction(tx); err != nil {
		panic(err)
	}

	stats := GetLedgerStats()
	fmt.Printf("UTXODB initialized:\nSeed: %s\nTotal supply = %di\nGenesis + %d predefined addresses with %di each\n",
		seedStr, supply, len(sigSchemes)-1, ownerAmount)

	fmt.Println("Balances:")
	for i, sigScheme := range sigSchemes {
		addr := sigScheme.Address()
		fmt.Printf("#%d: %s: balance %d, num outputs %d\n", i, addr.String(), stats[addr].Total, stats[addr].NumOutputs)
	}
}

func GetSupply() int64 {
	return supply
}

func GetGenesisSigScheme() signaturescheme.SignatureScheme {
	return sigSchemes[0]
}

func GetGenesisAddress() address.Address {
	return GetAddress(0)
}

func GetAddress(i int) address.Address {
	return sigSchemes[i].Address()
}

func GetSigScheme(addr address.Address) signaturescheme.SignatureScheme {
	return sigSchemesByAddress[addr]
}
