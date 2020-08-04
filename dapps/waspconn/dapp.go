package waspconn

import (
	"fmt"
	"net"
	"sync"

	"github.com/iotaledger/goshimmer/dapps/valuetransfers"
	"github.com/iotaledger/goshimmer/dapps/valuetransfers/packages/tangle"
	"github.com/iotaledger/goshimmer/dapps/valuetransfers/packages/transaction"
	"github.com/iotaledger/goshimmer/dapps/waspconn/packages/connector"
	"github.com/iotaledger/goshimmer/dapps/waspconn/packages/testing"
	"github.com/iotaledger/goshimmer/dapps/waspconn/packages/utxodb"
	"github.com/iotaledger/goshimmer/packages/shutdown"
	"github.com/iotaledger/goshimmer/plugins/config"
	"github.com/iotaledger/hive.go/daemon"
	"github.com/iotaledger/hive.go/events"
	"github.com/iotaledger/hive.go/logger"
	"github.com/iotaledger/hive.go/node"
	flag "github.com/spf13/pflag"
)

const (
	PluginName = "WaspConn"

	WaspConnPort          = "waspconn.port"
	WaspConnUtxodbEnabled = "waspconn.utxodbenabled"
)

func init() {
	flag.Int(WaspConnPort, 5000, "port for Wasp connections")
	flag.Bool(WaspConnUtxodbEnabled, true, "is utxodb mocking the value tangle enabled") // later change the default
}

var (
	app     *node.Plugin
	appOnce sync.Once

	PLUGINS = node.Plugins(
		App(),
	)
	log *logger.Logger

	emulator *utxodb.ConfirmEmulator
)

func App() *node.Plugin {
	appOnce.Do(func() {
		app = node.NewPlugin(PluginName, node.Enabled, configPlugin, runPlugin)
	})
	return app
}

func configPlugin(plugin *node.Plugin) {
	log = logger.NewLogger(PluginName)

	utxodbEnabled := config.Node().GetBool(WaspConnUtxodbEnabled)

	if utxodbEnabled {
		emulator = utxodb.NewConfirmEmulator()
		testing.Config(plugin, log, emulator)
		log.Infof("configured with UTXODB enabled")
	} else {
		configPluginStandard(plugin)
		log.Infof("configured for ValueTangle")
	}
}

func configPluginStandard(_ *node.Plugin) {
	valuetransfers.Tangle().Events.TransactionConfirmed.Attach(events.NewClosure(func(ctx *transaction.CachedTransaction, ctxMeta *tangle.CachedTransactionMetadata) {
		// TODO forward to connector.EventValueTransactionReceived
		tx := ctx.Unwrap() // ??
		if tx != nil {
			connector.EventValueTransactionReceived.Trigger(tx)
		}
	}))

	connector.EventValueTransactionReceived.Attach(events.NewClosure(func(tx *transaction.Transaction) {
		log.Debugf("EventValueTransactionReceived: txid = %s", tx.ID().String())
	}))

}

func runPlugin(_ *node.Plugin) {
	log.Debugf("starting WaspConn plugin on port %d", config.Node().GetInt(WaspConnPort))
	port := config.Node().GetInt(WaspConnPort)
	err := daemon.BackgroundWorker("WaspConn worker", func(shutdownSignal <-chan struct{}) {
		listenOn := fmt.Sprintf(":%d", port)
		listener, err := net.Listen("tcp", listenOn)
		if err != nil {
			log.Errorf("failed to start WaspConn daemon: %v", err)
			return
		}
		defer func() {
			_ = listener.Close()
		}()

		// TODO attach to goshimmer events

		go func() {
			// for each incoming connection spawns WaspConnector background worker
			for {
				conn, err := listener.Accept()
				if err != nil {
					return
				}
				log.Debugf("accepted connection from %s", conn.RemoteAddr().String())
				connector.Run(conn, log, emulator)
			}
		}()

		log.Debugf("running WaspConn plugin on port %d", port)

		<-shutdownSignal

		//log.Infof("Stopping WaspConn listener..")
		//_ = listener.Close()
		//log.Infof("Stopping WaspConn listener.. Done")
	}, shutdown.PriorityWaspConn)
	if err != nil {
		log.Errorf("failed to start WaspConn daemon: %v", err)
	}
}
