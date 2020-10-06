package drng

import (
	"encoding/json"

	flag "github.com/spf13/pflag"
)

const (
	// CfgDRNG defines the config flag of the DRNG.
	CfgDRNG = "drng"
)

func init() {
	flag.String(CfgDRNG, `"definitions":[{"instanceID":1, "threshold": 3, "distributedPubKey":"", "committeeMemebers":[]}]`, "dRNG default configuration")
}

type Definitions []definition

type definition struct {
	InstanceID        uint32
	Threshold         uint32
	DistributedPubKey string
	CommitteeMembers  []string
}

func parseCfg(cfg string) Definitions {
	var d Definitions
	json.Unmarshal([]byte(cfg), &d)
	return d
}
