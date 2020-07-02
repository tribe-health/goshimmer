package waspconn

import (
	"fmt"
	"github.com/iotaledger/goshimmer/dapps/waspconn/packages/connector"
	"net/http"

	"github.com/iotaledger/hive.go/events"

	"github.com/iotaledger/goshimmer/dapps/valuetransfers/packages/address"
	"github.com/iotaledger/goshimmer/dapps/valuetransfers/packages/transaction"
	"github.com/iotaledger/goshimmer/dapps/waspconn/packages/apilib"
	"github.com/iotaledger/goshimmer/dapps/waspconn/packages/utxodb"
	"github.com/iotaledger/goshimmer/plugins/gracefulshutdown"
	"github.com/iotaledger/goshimmer/plugins/webapi"
	"github.com/labstack/echo"
	"github.com/mr-tron/base58"
)

func addEndpoints() {
	webapi.Server().GET("/utxodb/outputs/:address", handleGetAddressOutputs)
	webapi.Server().POST("/utxodb/tx", handlePostTransaction)
	webapi.Server().GET("/adm/shutdown", handleShutdown)

	connector.EventValueTransactionReceived.Attach(events.NewClosure(func(tx *transaction.Transaction) {
		log.Debugf("EventValueTransactionReceived: txid = %s", tx.ID().String())
	}))
}

func handleGetAddressOutputs(c echo.Context) error {
	log.Debugw("handleGetAddressOutputs")
	addr, err := address.FromBase58(c.Param("address"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, &apilib.GetAccountOutputsResponse{Err: err.Error()})
	}
	outputs := utxodb.GetAddressOutputs(addr)
	log.Debugf("handleGetAddressOutputs: addr %s from utxodb %+v", addr.String(), outputs)

	out := make(map[string][]apilib.OutputBalance)
	for txOutId, txOutputs := range outputs {
		txOut := make([]apilib.OutputBalance, len(txOutputs))
		for i, txOutput := range txOutputs {
			txOut[i] = apilib.OutputBalance{
				Value: txOutput.Value,
				Color: transaction.ID(txOutput.Color).String(),
			}
		}
		out[txOutId.String()] = txOut
	}
	log.Debugw("handleGetAddressOutputs", "sending", out)

	return c.JSONPretty(http.StatusOK, &apilib.GetAccountOutputsResponse{
		Address: c.Param("address"),
		Outputs: out,
	}, " ")
}

func handlePostTransaction(c echo.Context) error {
	var req apilib.PostTransactionRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, &apilib.PostTransactionResponse{Err: err.Error()})
	}

	txBytes, err := base58.Decode(req.Tx)
	if err != nil {
		return c.JSON(http.StatusBadRequest, &apilib.PostTransactionResponse{Err: err.Error()})
	}

	tx, _, err := transaction.FromBytes(txBytes)
	if err != nil {
		return c.JSON(http.StatusBadRequest, &apilib.PostTransactionResponse{Err: err.Error()})
	}

	log.Debugf("handlePostTransaction:utxodb.AddTransaction: txid %s", tx.ID().String())

	err = utxodb.AddTransaction(tx)
	if err != nil {
		log.Warnf("handlePostTransaction:utxodb.AddTransaction: txid %s", tx.ID().String())
		return c.JSON(http.StatusConflict, &apilib.PostTransactionResponse{Err: err.Error()})
	}

	connector.EventValueTransactionReceived.Trigger(tx)

	return c.JSON(http.StatusOK, &apilib.PostTransactionResponse{})
}

func handleShutdown(c echo.Context) error {
	gracefulshutdown.ShutdownWithError(fmt.Errorf("Shutdown requested from WebAPI."))
	return nil
}
