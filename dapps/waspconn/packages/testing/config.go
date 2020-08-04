package testing

import (
	"github.com/iotaledger/goshimmer/dapps/waspconn/packages/utxodb"
	"github.com/iotaledger/hive.go/logger"
	"github.com/iotaledger/hive.go/node"
)

var log *logger.Logger

func Config(_ *node.Plugin, setLog *logger.Logger, emulator *utxodb.ConfirmEmulator) {
	log = setLog.Named("testing")
	addEndpoints(emulator)
}
