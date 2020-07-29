package testing

import (
	"github.com/iotaledger/goshimmer/dapps/waspconn/packages/utxodb"
	"github.com/iotaledger/goshimmer/plugins/config"
	"github.com/iotaledger/hive.go/logger"
	"github.com/iotaledger/hive.go/node"
	flag "github.com/spf13/pflag"
	"time"
)

const (
	WaspConnUtxodbConfirmDelay           = "waspconn.utxodbconfirmseconds"
	WaspConnUtxodbConfirmRandomize       = "waspconn.utxodbconfirmrandomize"
	WaspConnUtxodbConfirmFirstInConflict = "waspconn.utxodbconfirmfirst"
)

var log *logger.Logger

func Config(_ *node.Plugin, setLog *logger.Logger) {
	log = setLog.Named("testing")

	flag.Int(WaspConnUtxodbConfirmDelay, 0, "emulated confirmation delay for utxodb in seconds")
	flag.Bool(WaspConnUtxodbConfirmRandomize, false, "is confirmation time random with the mean at confirmation delay")
	flag.Bool(WaspConnUtxodbConfirmFirstInConflict, false, "in case of conflict, confirm first transaction. Default is reject all")

	confDelay := time.Duration(config.Node().GetInt(WaspConnUtxodbConfirmDelay)) * time.Second
	randomize := config.Node().GetBool(WaspConnUtxodbConfirmRandomize)
	confirmFirst := config.Node().GetBool(WaspConnUtxodbConfirmFirstInConflict)

	utxodb.SetConfirmationParams(confDelay, randomize, confirmFirst)

	log.Infof("UTXODB confirmation delay (mean): %v, randomize confirmation: %v, confirm first: %v ",
		confDelay, randomize, confirmFirst)

	addEndpoints()
}
