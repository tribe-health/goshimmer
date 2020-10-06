package committee

import (
	"encoding/hex"
	"net/http"

	"github.com/iotaledger/goshimmer/plugins/drng"
	"github.com/iotaledger/hive.go/crypto/ed25519"
	"github.com/labstack/echo"
	"github.com/mr-tron/base58"
)

// Handler returns the current DRNG committee used.
func Handler(c echo.Context) error {
	committees := []Committee{}
	for _, state := range drng.Instance().State {
		committees = append(committees, Committee{
			InstanceID:    state.Committee().InstanceID,
			Threshold:     state.Committee().Threshold,
			Identities:    identitiesToString(state.Committee().Identities),
			DistributedPK: hex.EncodeToString(state.Committee().DistributedPK),
		})
	}

	return c.JSON(http.StatusOK, Response{
		Committees: committees,
	})
}

// Response is the HTTP message containing the DRNG committee.
type Response struct {
	Committees []Committee `json:"committees,omitempty"`
	Error      string      `json:"error,omitempty"`
}

type Committee struct {
	InstanceID    uint32   `json:"instanceID,omitempty"`
	Threshold     uint8    `json:"threshold,omitempty"`
	Identities    []string `json:"identities,omitempty"`
	DistributedPK string   `json:"distributedPK,omitempty"`
}

func identitiesToString(publicKeys []ed25519.PublicKey) []string {
	identities := []string{}
	for _, pk := range publicKeys {
		identities = append(identities, base58.Encode(pk[:]))
	}
	return identities
}
