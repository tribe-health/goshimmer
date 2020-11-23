package pow

import (
	"fmt"
	"net/http"
	"sync"

	"github.com/iotaledger/goshimmer/plugins/pow"
	"github.com/iotaledger/goshimmer/plugins/webapi"
	"github.com/iotaledger/hive.go/node"
	"github.com/labstack/echo"
)

// PluginName is the name of the web API PoW endpoint plugin.
const PluginName = "WebAPI PoW Endpoint"

var (
	// plugin is the plugin instance of the web API PoW endpoint plugin.
	plugin *node.Plugin
	once   sync.Once
)

func configure(plugin *node.Plugin) {
	webapi.Server().GET("pow/tune", handler)
}

// Plugin gets the plugin instance.
func Plugin() *node.Plugin {
	once.Do(func() {
		plugin = node.NewPlugin(PluginName, node.Enabled, configure)
	})
	return plugin
}

// tune sets the pow difficulty
func handler(c echo.Context) error {
	var request Request
	if err := c.Bind(&request); err != nil {
		return c.JSON(http.StatusBadRequest, Response{Error: err.Error()})
	}

	pow.Tune(request.Difficulty)

	return c.JSON(http.StatusOK, Response{Message: fmt.Sprintf("PoW difficulty changed to %d", request.Difficulty)})
}

// Response is the HTTP response of a pow tune request.
type Response struct {
	Message string `json:"message"`
	Error   string `json:"error"`
}

// Request contains the parameters of a pow tune request.
type Request struct {
	Difficulty int `json:"difficulty"`
}
