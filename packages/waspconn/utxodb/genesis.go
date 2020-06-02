package utxodb

import (
	"bytes"
	"fmt"
	"github.com/iotaledger/goshimmer/dapps/valuetransfers/packages/address"
	"github.com/iotaledger/goshimmer/dapps/valuetransfers/packages/address/signaturescheme"
	"github.com/iotaledger/goshimmer/dapps/valuetransfers/packages/balance"
	"github.com/iotaledger/goshimmer/dapps/valuetransfers/packages/transaction"
	"github.com/iotaledger/hive.go/crypto/ed25519"
	"github.com/mr-tron/base58"
	"sort"
)

const (
	genesisPrivateKeyStr = "1pK9KraR4YTSHh3bq7hrigFSyq4HgWufhRyME84DPbwWpcoF1zwq6J1zaeyYUb8ut6ia9uQ9B9ughrpj2aZ7CMU"
	genesisPublicKeyStr  = "4f1W4o6PBKXXFsMHRcndYabpmaXPTdNNmtnV2NkfHnAa"
	supply               = int64(100 * 1000 * 1000 * 1000)
	ownerAmount          = 1000 * 1000 * 1000
)

type keyData struct {
	priKey  string
	pubKey  string
	address string
}

// hardcoded addresses
var keyStr = []keyData{
	{"2sHwx4pBsAxK5SV8Nurphsgu6XpRE2ZpBxJQ87ggdoU34NAQ5RPM6u9K8oL4UdbYgv1pRiHNHCsWipbY1TvEuXz6",
		"C3XgXzR6dVCsnXYqxJC5WgMEQJUDqeChUMKY9Tuvfoux",
		"b1uERj4r1wuuVeTx4n7t8JnsuSF5YnZPf8CQdCzv6EHs",
	},
	{"6788yntE9kZeg9H7VLyG9F7pjTMpmQm3MbtSGVgcYKQ9AmxpwJf6oZsiuiDb6qMwpEJeofT28M1gu2c7iqKPCgag",
		"AsYBP4Yd41e3noTx9rzZKekwnZ1wkPokgb7i7RKKdUKN",
		"VSVggSWMLdT5xcXpyd1yQAJKMKzEwqP2Ff65KHFbCzjD",
	},
	{"2CESGXeY11kmRec7et5ALBeDDKzyG7YEQXbVu4Z1XYiVeeZbYJSrrhtLuiggk4EfGWeFc2G33kWeeHTuhoGk9dXU",
		"8GrpoaDAV3om7Ufa8frapCfDr6pprN3zsBNzdh8uFxKg",
		"S5ugKUKvntBi66g44qGTa3jcSgDh6QxYHgAhJ5s5m61x",
	},
	{"44nD5o7cBjsjHeqVV7D5R9Qj1MBtQSY6ad15LsCNKgeEC1QBina4Rx8hivdy424mTJ2z7wRHKFQBJgmAm5nkpYvn",
		"5e8C1cNyKbwHgtCNzyENMfq8nuhkye6baW9uiTwAi7U6",
		"L2eQY4VP22sZztFiRyAN1XrSkBLf83pveUzp5QwKyGMf",
	},
	{"3h1MBDT7FrvArhiGNtEFHLsQdpsEq89wKVuDTQSQDqKK2Y4nuGHabjDKnPTBN5L5w7g6gzGu99zCxsVgxm2LT14X",
		"9k1bCVfk3Xd6mPhi7ABxGR12AYtqsn7gVxp5EaHkR24j",
		"RmhH2JNGA87mvHbrmyM25fFsTXgmFbq9iLVtpsr5x99P",
	},
	{"4emvjETw6TUdywhoLhPduNqUmtGmqmyqPcdLC6sj3tsVR5rekdMr47pYkBoL9VY3Vn3UjgFzSXnWgru5sfQcLw4Q",
		"HHDctktieZdFwjmkJDicawruc8qEHJuPfpjFVN7UwA12",
		"PRTdqHcYJWbCuNTNQk1ZEjCpYoVYcM8ehcU6w5uCB1po",
	},
	{"36hT9rBbbrJ1tR8i7LwQMGR3AWsQFM2XVxNx7CbFgpibTc4Naz9VyaFVwoHMLBmK93QzQny73NAWHNoej8oqA8er",
		"B3pNXn5298PmCjCECMhGuX2XqZy6pnqPw9JZb65cQxPx",
		"X6CipXF3BKzZwE5i6nJgASMBUeYy67kC2ThNeMt6V2Hz",
	},
	{"62zFB4cNGNLEi48ZuMuXGTt7g9bUEd17JqX6HWQHN4SsBmTQWjQHvCsUmaL29bAAB59XYdA5DnhT9fjzumnAZ7c",
		"Aasc6GQEUsv7RBxhu7kRGSzoeYSFD6zu8VrjYM5rvHMQ",
		"JjtsUvuvDHainJp7b4RXqCNXFpGeNZCcSF46BdLY5ewX",
	},
	{"4EkU5euz1MjQYrw7Vge8fi7w43kzSvaSaVWQqHpt96rbL5gydtkmAKntLB7WuoFVRqeLXs2MXwf4pWwMtU3cqZG4",
		"G13F4WWutdLoyrPkrzjYkYJMUgmBnNEhSz7yQPoD6QME",
		"NpETKpMAjRnLqqeNBGZJw1CeDeCp7PCq8osrVrvwozBv",
	},
	{"39ZMAioHL74xs9wwznMxVJCEtxFxqVBpi7hGzAwAvUm6jQfRBq2eQRWoKsZjLeXFky3RGuv8NGzUsj4vg9Z9g8X7",
		"2793HqRiS5vdTa2RbqcGi9phmMHYkuax1ALzs1qc9bCq",
		"ZJ9MV5hKYcrpkRtpN8ePb1JeamsocZ7tcfm325tsY66R",
	},
}

var (
	genesisTxId transaction.ID

	knownAddresses  []address.Address
	knownSigSchemes map[address.Address]signaturescheme.SignatureScheme
)

func init() {
	knownAddresses = make([]address.Address, len(keyStr)+1)
	knownSigSchemes = make(map[address.Address]signaturescheme.SignatureScheme)

	genesis := createSigScheme(genesisPrivateKeyStr, genesisPublicKeyStr, "")
	knownAddresses[0] = genesis.Address()
	knownSigSchemes[genesis.Address()] = genesis

	for i := 0; i < len(knownAddresses)-1; i++ {
		sigs := createSigScheme(keyStr[i].priKey, keyStr[i].pubKey, keyStr[i].address)
		knownSigSchemes[sigs.Address()] = sigs
		knownAddresses[i+1] = sigs.Address()
	}
	// create genesis transaction

	var niltxid transaction.ID

	genesisInput := transaction.NewOutputID(GetGenesisAddress(), niltxid)
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

	testAddresses := make([]address.Address, len(knownAddresses)-1)
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
	fmt.Printf("UTXODB initialized:\nTotal supply = %di\nGenesis + %d predefined addresses with %di each\n",
		supply, len(knownAddresses)-1, ownerAmount)
	sorted := make([]address.Address, 0, len(stats))
	for a := range stats {
		sorted = append(sorted, a)
	}
	sort.Slice(sorted, func(i, j int) bool {
		return bytes.Compare(sorted[i][:], sorted[j][:]) < 0
	})

	fmt.Println("Balances:")
	for _, addr := range sorted {
		fmt.Printf("%s: balance %d, num outputs %d\n", addr.String(), stats[addr].Total, stats[addr].NumOutputs)
	}
}

func GetSupply() int64 {
	return supply
}

func GetGenesisSigScheme() signaturescheme.SignatureScheme {
	return knownSigSchemes[GetGenesisAddress()]
}

//
//func GetGenesisTransaction() *transaction.Transaction {
//	ret, ok := GetTransaction(genesisTxId)
//	if !ok {
//		panic("genesis tx not found")
//	}
//	return ret
//}

func GetGenesisAddress() address.Address {
	return GetAddress(0)
}

func GetAddress(i int) address.Address {
	return knownAddresses[i]
}

func GetSigScheme(addr address.Address) signaturescheme.SignatureScheme {
	return knownSigSchemes[addr]
}

func createSigScheme(privKeyStr, pubKeyStr, addressStr string) signaturescheme.SignatureScheme {
	var privKey ed25519.PrivateKey
	var pubKey ed25519.PublicKey
	var err error

	priv, err := base58.Decode(privKeyStr)
	if err != nil || len(priv) != len(privKey) {
		panic(err)
	}
	pub, err := base58.Decode(pubKeyStr)
	if err != nil || len(pub) != len(pubKey) {
		panic(err)
	}
	copy(privKey[:], priv)
	copy(pubKey[:], pub)
	keyPair := ed25519.KeyPair{
		PrivateKey: privKey,
		PublicKey:  pubKey,
	}
	ret := signaturescheme.ED25519(keyPair)

	if addressStr == "" {
		return ret
	}
	addr, err := address.FromBase58(addressStr)
	if err != nil {
		panic(err)
	}
	if addr != ret.Address() {
		panic("addr != ret.Address()")
	}
	return ret
}
