package waspconn

import (
	"flag"
	"fmt"
	"github.com/iotaledger/goshimmer/dapps/waspconn/packages/connector"
	"github.com/iotaledger/goshimmer/dapps/waspconn/packages/utxodb"
	"github.com/iotaledger/goshimmer/packages/shutdown"
	"github.com/iotaledger/goshimmer/plugins/config"
	"github.com/iotaledger/hive.go/daemon"
	"github.com/iotaledger/hive.go/logger"
	"github.com/iotaledger/hive.go/node"
	"net"
	"sync"
	"time"
)

const (
	PluginName                 = "WaspConn"
	WaspConnPort               = "waspconn.port"
	WaspConnUtxodbConfirmDelay = "waspconn.utxodbconfirmseconds"
)

var (
	app     *node.Plugin
	appOnce sync.Once

	PLUGINS = node.Plugins(
		App(),
	)
	log *logger.Logger
)

func App() *node.Plugin {
	appOnce.Do(func() {
		app = node.NewPlugin(PluginName, node.Enabled, configPlugin, runPlugin)
	})
	return app
}

func configPlugin(_ *node.Plugin) {
	log = logger.NewLogger(PluginName)

	flag.Int(WaspConnPort, 5000, "port for Wasp connections")
	flag.Int(WaspConnUtxodbConfirmDelay, 0, "emulated confirmation delay for utxodb in seconds")
	confDelay := time.Duration(config.Node().GetInt(WaspConnUtxodbConfirmDelay)) * time.Second
	utxodb.SetConfirmationTime(confDelay)

	log.Infof("UTXODB confirmation delay: %v", confDelay)

	addEndpoints()
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
				connector.Run(conn, log)
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
