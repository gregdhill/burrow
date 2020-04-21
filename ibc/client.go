package ibc

import (
	"time"

	clientexported "github.com/cosmos/cosmos-sdk/x/ibc/02-client/exported"
	connectionexported "github.com/cosmos/cosmos-sdk/x/ibc/03-connection/exported"
	channelexported "github.com/cosmos/cosmos-sdk/x/ibc/04-channel/exported"
	commitmentexported "github.com/cosmos/cosmos-sdk/x/ibc/23-commitment/exported"
	"github.com/tendermint/go-amino"
)

// https://github.com/cosmos/ics/tree/master/spec/ics-002-client-semantics

var _ clientexported.ClientState = &ClientState{}

type ClientState struct {
	id        string
	chainID   string
	frozen    bool
	trusting  time.Duration
	unbonding time.Duration
	last      Header
}

func NewClientState() *ClientState {
	return &ClientState{}
}

func (cs *ClientState) GetID() string {
	return cs.id
}

func (cs *ClientState) GetChainID() string {
	return cs.chainID
}

func (cs *ClientState) ClientType() clientexported.ClientType {
	return clientexported.Tendermint
}

func (cs *ClientState) GetLatestHeight() uint64 {
	return 0
}

func (cs *ClientState) IsFrozen() bool {
	return cs.frozen
}

func (cs *ClientState) VerifyClientConsensusState(
	cdc *amino.Codec,
	root commitmentexported.Root,
	height uint64,
	counterpartyClientIdentifier string,
	consensusHeight uint64,
	prefix commitmentexported.Prefix,
	proof commitmentexported.Proof,
	consensusState clientexported.ConsensusState,
) error {
	return nil
}

func (cs *ClientState) VerifyConnectionState(
	cdc *amino.Codec,
	height uint64,
	prefix commitmentexported.Prefix,
	proof commitmentexported.Proof,
	connectionID string,
	connectionEnd connectionexported.ConnectionI,
	consensusState clientexported.ConsensusState,
) error {
	return nil
}

func (cs *ClientState) VerifyChannelState(
	cdc *amino.Codec,
	height uint64,
	prefix commitmentexported.Prefix,
	proof commitmentexported.Proof,
	portID,
	channelID string,
	channel channelexported.ChannelI,
	consensusState clientexported.ConsensusState,
) error {
	return nil
}

func (cs *ClientState) VerifyPacketCommitment(
	height uint64,
	prefix commitmentexported.Prefix,
	proof commitmentexported.Proof,
	portID,
	channelID string,
	sequence uint64,
	commitmentBytes []byte,
	consensusState clientexported.ConsensusState,
) error {
	return nil
}

func (cs *ClientState) VerifyPacketAcknowledgement(
	height uint64,
	prefix commitmentexported.Prefix,
	proof commitmentexported.Proof,
	portID,
	channelID string,
	sequence uint64,
	acknowledgement []byte,
	consensusState clientexported.ConsensusState,
) error {
	return nil
}

func (cs *ClientState) VerifyPacketAcknowledgementAbsence(
	height uint64,
	prefix commitmentexported.Prefix,
	proof commitmentexported.Proof,
	portID,
	channelID string,
	sequence uint64,
	consensusState clientexported.ConsensusState,
) error {
	return nil
}

func (cs *ClientState) VerifyNextSequenceRecv(
	height uint64,
	prefix commitmentexported.Prefix,
	proof commitmentexported.Proof,
	portID,
	channelID string,
	nextSequenceRecv uint64,
	consensusState clientexported.ConsensusState,
) error {
	return nil
}
