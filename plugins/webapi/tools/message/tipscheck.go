package message

import (
	"net/http"

	"github.com/iotaledger/goshimmer/packages/tangle"
	"github.com/iotaledger/goshimmer/plugins/messagelayer"
	"github.com/iotaledger/hive.go/objectstorage"
	"github.com/labstack/echo"
)

// TipsCheckHandler process tipscheck requests.
func TipsCheckHandler(c echo.Context) error {
	res := &TipsCheckResponse{}
	tips := messagelayer.TipSelector().AllTips()
	for _, tip := range tips {
		tipStatus := TipStatus{ID: tip.String(), ApproversSolid: true}
		messagelayer.Tangle().MessageMetadata(tip).Consume(func(object objectstorage.StorableObject) {
			metadata := object.(*tangle.MessageMetadata)
			if metadata != nil {
				tipStatus.Solid = metadata.IsSolid()
			}
		})

		cachedApprovers := messagelayer.Tangle().Approvers(tip)
		if len(cachedApprovers) == 0 {
			tipStatus.Status = true
		}
		cachedApprovers.Consume(func(approver *tangle.Approver) {
			messagelayer.Tangle().MessageMetadata(approver.ApproverMessageID()).Consume(func(object objectstorage.StorableObject) {
				metadata := object.(*tangle.MessageMetadata)
				if metadata != nil {
					tipStatus.ApproversSolid = tipStatus.ApproversSolid && metadata.IsSolid()
				}
			}, false)
		})

		res.Tips = append(res.Tips, tipStatus)
	}

	return c.JSON(http.StatusOK, res)
}

// TipsCheckResponse is the HTTP response containing all the current Tips and their status.
type TipsCheckResponse struct {
	Tips []TipStatus `json:"tips,omitempty"`
}

type TipStatus struct {
	ID             string `json:"ids"`
	Status         bool   `json:"status"`
	Solid          bool   `json:"solid"`
	ApproversSolid bool   `json:"approvers_solid"`
}
