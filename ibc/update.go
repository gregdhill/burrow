package ibc

import (
	"time"

	lite "github.com/tendermint/tendermint/lite2"
	tmtypes "github.com/tendermint/tendermint/types"
)

type Header struct {
	tmtypes.SignedHeader
	ValidatorSet *tmtypes.ValidatorSet
}

func (cs *ClientState) CheckValidityAndUpdateState(header Header) error {
	err := lite.Verify(cs.GetChainID(), &cs.last.SignedHeader, cs.last.ValidatorSet,
		&header.SignedHeader, header.ValidatorSet, cs.trusting, time.Now(), lite.DefaultTrustLevel)
	if err != nil {
		return err
	}

	cs.last = header
	return nil
}
