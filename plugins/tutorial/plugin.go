package tutorial

import (
	"sync"

	"github.com/iotaledger/goshimmer/packages/tangle"
	"github.com/iotaledger/goshimmer/plugins/messagelayer"
	"github.com/iotaledger/goshimmer/plugins/pow"
	"github.com/iotaledger/hive.go/events"
	"github.com/iotaledger/hive.go/logger"
	"github.com/iotaledger/hive.go/node"
)

// PluginName is the name of the tutorial plugin.
const PluginName = "tutorial"

var (
	// plugin is the plugin instance of the web API plugin.
	plugin     *node.Plugin
	pluginOnce sync.Once

	// log is the logger used by this plugin.
	log *logger.Logger
)

// Plugin gets the plugin instance.
func Plugin() *node.Plugin {
	pluginOnce.Do(func() {
		plugin = node.NewPlugin(PluginName, node.Disabled, configure, run)
	})
	return plugin
}

func configure(*node.Plugin) {
	log = logger.NewLogger(PluginName)
	messagelayer.Tangle().Events.MessageSolid.Attach(events.NewClosure(func(cachedMsgEvent *tangle.CachedMessageEvent) {
		defer cachedMsgEvent.MessageMetadata.Release()
		cachedMsgEvent.Message.Consume(func(message *tangle.Message) {
			log.Info(message.IssuingTime().UnixNano(), message.ID(), "-->", message.StrongParents(), message.WeakParents())
		})
	}))

	pow.Events().PowDone.Attach(events.NewClosure(func(ev *pow.PowDoneEvent) {
		log.Info(ev.Difficulty, ev.Duration)
	}))
}

func run(*node.Plugin) {
	log.Infof("Starting %s ...", PluginName)
}
