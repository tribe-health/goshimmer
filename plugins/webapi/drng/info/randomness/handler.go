package randomness

import (
	"net/http"
	"time"

	"github.com/iotaledger/goshimmer/plugins/drng"
	"github.com/labstack/echo"
)

// Handler returns the current DRNG randomness used.
func Handler(c echo.Context) error {
	randomness := []Randomness{}
	for _, state := range drng.Instance().State {
		randomness = append(randomness,
			Randomness{
				InstanceID: state.Committee().InstanceID,
				Round:      state.Randomness().Round,
				Randomness: state.Randomness().Randomness,
				Timestamp:  state.Randomness().Timestamp,
			})
	}

	return c.JSON(http.StatusOK, Response{
		Randomness: randomness,
	})
}

// Response is the HTTP message containing the current DRNG randomness.
type Response struct {
	Randomness []Randomness `json:"randomness,omitempty"`
	Error      string       `json:"error,omitempty"`
}

type Randomness struct {
	InstanceID uint32    `json:"instanceID,omitempty"`
	Round      uint64    `json:"round,omitempty"`
	Timestamp  time.Time `json:"timestamp,omitempty"`
	Randomness []byte    `json:"randomness,omitempty"`
}
