package drng

import (
	"encoding/hex"
	"errors"
	"fmt"

	"github.com/iotaledger/goshimmer/packages/binary/drng"
	"github.com/iotaledger/goshimmer/packages/binary/drng/state"
	cbPayload "github.com/iotaledger/goshimmer/packages/binary/drng/subtypes/collectivebeacon/payload"
	"github.com/iotaledger/goshimmer/plugins/config"
	"github.com/iotaledger/hive.go/crypto/ed25519"
	"github.com/iotaledger/hive.go/logger"
	"github.com/mr-tron/base58/base58"
)

var (
	// ErrParsingCommitteeMember is returned for an invalid committee member
	ErrParsingCommitteeMember = errors.New("cannot parse committee member")
)

func configureDRNG() *drng.DRNG {
	log = logger.NewLogger(PluginName)
	configuration := make(map[uint32][]state.Option)

	committees := parseCfg(config.Node().GetString(CfgDRNG))

	for _, committee := range committees {
		// parse identities of the committee members
		committeeMembers, err := parseCommitteeMembers(committee.CommitteeMembers)
		if err != nil {
			log.Warnf("Invalid %s: %s", committee.CommitteeMembers, err)
		}

		// parse distributed public key of the committee
		var dpk []byte
		if committee.DistributedPubKey != "" {
			bytes, err := hex.DecodeString(committee.DistributedPubKey)
			if err != nil {
				log.Warnf("Invalid %s: %s", committee.DistributedPubKey, err)
			}
			if l := len(bytes); l != cbPayload.PublicKeySize {
				log.Warnf("Invalid %s length: %d, need %d", committee.DistributedPubKey, l, cbPayload.PublicKeySize)
			}
			dpk = append(dpk, bytes...)
		}

		// configure committee
		committeeConf := &state.Committee{
			InstanceID:    committee.InstanceID,
			Threshold:     uint8(committee.Threshold),
			DistributedPK: dpk,
			Identities:    committeeMembers,
		}

		configuration[committee.InstanceID] = []state.Option{state.SetCommittee(committeeConf)}
	}

	return drng.New(configuration)
}

// Instance returns the DRNG instance.
func Instance() *drng.DRNG {
	once.Do(func() { instance = configureDRNG() })
	return instance
}

func parseCommitteeMembers(members []string) (result []ed25519.PublicKey, err error) {
	for _, committeeMember := range members {
		if committeeMember == "" {
			continue
		}

		pubKey, err := base58.Decode(committeeMember)
		if err != nil {
			return nil, fmt.Errorf("%w: invalid public key: %s", ErrParsingCommitteeMember, err)
		}
		publicKey, _, err := ed25519.PublicKeyFromBytes(pubKey)
		if err != nil {
			return nil, err
		}

		result = append(result, publicKey)
	}

	return result, nil
}
