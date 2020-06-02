package waspconn

import (
	"flag"
	"fmt"
	"github.com/iotaledger/goshimmer/dapps/valuetransfers/packages/transaction"
	"github.com/iotaledger/goshimmer/packages/shutdown"
	"github.com/iotaledger/goshimmer/plugins/config"
	"github.com/iotaledger/hive.go/daemon"
	"github.com/iotaledger/hive.go/events"
	"github.com/iotaledger/hive.go/logger"
	"github.com/iotaledger/hive.go/node"
	"net"
)

const (
	name         = "WaspConn"
	WaspConnPort = "waspconn.port"
)

var (
	PLUGIN                        = node.NewPlugin(name, node.Enabled, configPlugin, runPlugin)
	PLUGINS                       = node.Plugins(PLUGIN)
	EventValueTransactionReceived *events.Event
	log                           *logger.Logger
)

func configPlugin(_ *node.Plugin) {
	log = logger.NewLogger(name)

	flag.Int(WaspConnPort, 5000, "port for Wasp connections")

	EventValueTransactionReceived = events.NewEvent(func(handler interface{}, params ...interface{}) {
		handler.(func(_ *transaction.Transaction))(params[0].(*transaction.Transaction))
	})
}

func runPlugin(_ *node.Plugin) {
	log.Debugf("starting WaspConn plugin on port %d", config.Node.GetInt(WaspConnPort))
	port := config.Node.GetInt(WaspConnPort)
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
				Run(conn, log)
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
