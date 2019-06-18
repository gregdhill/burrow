// +build forensics

package forensics

import (

	"context"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"testing"

	"github.com/hyperledger/burrow/integration/rpctest"
	"github.com/hyperledger/burrow/rpc/rpcevents"
	"github.com/stretchr/testify/assert"
	"github.com/hyperledger/burrow/config/source"
	"github.com/hyperledger/burrow/execution/state"
	"github.com/hyperledger/burrow/genesis"
	"github.com/hyperledger/burrow/logging"
	"github.com/stretchr/testify/require"
	"github.com/sergi/go-diff/diffmatchpatch"
	
)

// This serves as a testbed for looking at non-deterministic burrow instances capture from the wild
// Put the path to 'good' and 'bad' burrow directories here (containing the config files and .burrow dir)
//const goodDir = "/home/silas/test-chain"
const badDir = "/home/greg/go/src/github.com/hyperledger/burrow/.state001"
const goodDir = "/home/greg/go/src/github.com/hyperledger/burrow/.state002"
const genFile = "/home/greg/go/src/github.com/hyperledger/burrow/t9-gen.json"
const criticalBlock uint64 = 52

func TestReplay_Compare(t *testing.T) {
	badReplay := newReplay(t, badDir)
	goodReplay := newReplay(t, goodDir)
	badRecaps, err := badReplay.Blocks(2, criticalBlock+1)
	require.NoError(t, err)
	goodRecaps, err := goodReplay.Blocks(2, criticalBlock+1)
	require.NoError(t, err)
	for i, goodRecap := range goodRecaps {
		fmt.Printf("Good: %v\n", goodRecap)
		fmt.Printf("Bad: %v\n", badRecaps[i])
		assert.Equal(t, goodRecap, badRecaps[i])
		for i, txe := range goodRecap.TxExecutions {
			fmt.Printf("Tx %d: %v\n", i, txe.TxHash)
			fmt.Println(txe.Envelope)
		}
		fmt.Println()
	}

	txe := goodRecaps[5].TxExecutions[0]
	assert.Equal(t, badRecaps[5].TxExecutions[0], txe)
	fmt.Printf("%v \n\n", txe)

	cli := rpctest.NewExecutionEventsClient(t, "localhost:10997")
	txeRemote, err := cli.Tx(context.Background(), &rpcevents.TxRequest{
		TxHash: txe.TxHash,
	})
	require.NoError(t, err)
	err = ioutil.WriteFile("txe.json", []byte(source.JSONString(txe)), 0600)
	require.NoError(t, err)
	err = ioutil.WriteFile("txeRemote.json", []byte(source.JSONString(txeRemote)), 0600)
	require.NoError(t, err)

	fmt.Println(txeRemote)
}

func TestDecipher(t *testing.T) {
	hexmsg:= "7B22436861696E4944223A2270726F64756374696F6E2D74392D73747564696F2D627572726F772D364337333335222C2254797065223A2243616C6C5478222C225061796C6F6164223A7B22496E707574223A7B2241646472657373223A2236354139334431443333423633453932453942454335463938444633313638303033384530303431222C2253657175656E6365223A34307D2C2241646472657373223A2242413544333042313031393233363033444331333133313231334431334633443939354138344142222C224761734C696D6974223A393939393939392C2244617461223A224636373138374143303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303032303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030304534343635363136343643363936453635344637323631363336433635303030303030303030303030303030303030303030303030303030303030303030303030222C225741534D223A22227D7D"
	bs, err := hex.DecodeString(hexmsg)
	require.NoError(t, err)
	fmt.Println(string(bs))
}

func TestReplay_Good(t *testing.T) {
	replay := newReplay(t, goodDir)
	_, err := replay.Blocks(1, criticalBlock+1)
	require.NoError(t, err)
}

func TestReplay_Bad(t *testing.T) {
	replay := newReplay(t, badDir)
	_, err := replay.Blocks(1, criticalBlock+1)
	require.NoError(t, err)
}

func TestStateHashes_Bad(t *testing.T) {
	badReplay := newReplay(t, badDir)
	goodReplay := newReplay(t, goodDir)
	for i := uint64(0); i <= criticalBlock+1; i++ {
		fmt.Println("Good")
		goodSt, err := goodReplay.State(i)
		require.NoError(t, err)
		fmt.Printf("Good: Version: %d, Hash: %X\n", goodSt.Version(), goodSt.Hash())
		fmt.Println("Bad")
		badSt, err := badReplay.State(i)
		require.NoError(t, err)
		fmt.Printf("Bad: Version: %d, Hash: %X\n", badSt.Version(), badSt.Hash())
		fmt.Println()
	}
}

func TestReplay_Good_Block(t *testing.T) {
	replayBlock(t, goodDir, criticalBlock)
}

func TestReplay_Bad_Block(t *testing.T) {
	replayBlock(t, badDir, criticalBlock)
}

func TestCriticalBlock_CommitTree(t *testing.T) {
	// go test ./forensics -tags=forensics -v -run TestCriticalBlock_CommitTree

	badState := getState(t, badDir, criticalBlock)
	goodState := getState(t, goodDir, criticalBlock)

	badCommitTree := badState.DumpCommits()
	goodCommitTree := goodState.DumpCommits()

	if badCommitTree != goodCommitTree {
		dmp := diffmatchpatch.New()
		diffs := dmp.DiffMain(badCommitTree, goodCommitTree, true)
		fmt.Println(dmp.DiffPrettyText(diffs))
		assert.Fail(t, "commits trees do not match")
	}
}

func TestReplay_BadCommits(t *testing.T) {
	// go test ./forensics -tags=forensics -v -run TestReplay_BadCommits

	fmt.Println("Good >")
	replay := newReplay(t, goodDir)
	replay.Blocks(1, criticalBlock+1)

	fmt.Println("Bad >")
	replay = newReplay(t, badDir)
	replay.Blocks(1, criticalBlock+1)
}

func replayBlock(t *testing.T, burrowDir string, height uint64) {
	replay := newReplay(t, burrowDir)
	//replay.State()
	recap, err := replay.Block(height)
	require.NoError(t, err)
	recap.TxExecutions = nil
	fmt.Println(recap)
}

func getState(t *testing.T, burrowDir string, height uint64) *state.State {
	st, err := newReplay(t, burrowDir).State(height)
	require.NoError(t, err)
	return st
}

func newReplay(t *testing.T, burrowDir string) *Replay {
	genesisDoc := new(genesis.GenesisDoc)
	err := source.FromFile(genFile, genesisDoc)
	require.NoError(t, err)
	return NewReplay(burrowDir, genesisDoc, logging.NewNoopLogger())
}
