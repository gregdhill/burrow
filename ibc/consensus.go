package ibc

import (
	clientexported "github.com/cosmos/cosmos-sdk/x/ibc/02-client/exported"
	commitmentexported "github.com/cosmos/cosmos-sdk/x/ibc/23-commitment/exported"
)

var _ clientexported.ConsensusState = &ConsensusState{}

type ConsensusState struct {
}

func NewConsensusState() *ConsensusState {
	return &ConsensusState{}
}

func (cs *ConsensusState) ClientType() clientexported.ClientType {
	return clientexported.Tendermint
}

func (cs *ConsensusState) GetHeight() uint64 {
	return 0
}

func (cs *ConsensusState) GetRoot() commitmentexported.Root {
	return nil
}

func (cs *ConsensusState) ValidateBasic() error {
	return nil
}
