// +build forensics

package forensics

import (
	"bytes"
	"fmt"
	"path"

	"github.com/hyperledger/burrow/bcm"
	"github.com/hyperledger/burrow/binary"
	"github.com/hyperledger/burrow/core"
	"github.com/hyperledger/burrow/event"
	"github.com/hyperledger/burrow/execution"
	"github.com/hyperledger/burrow/execution/exec"
	"github.com/hyperledger/burrow/execution/state"
	"github.com/hyperledger/burrow/genesis"
	"github.com/hyperledger/burrow/logging"
	"github.com/hyperledger/burrow/storage"
	"github.com/hyperledger/burrow/txs"
	"github.com/pkg/errors"
	dbm "github.com/tendermint/tendermint/libs/db"
	"github.com/tendermint/tendermint/types"
)

type Replay struct {
	explorer   *bcm.BlockStore
	db         dbm.DB
	cacheDB    dbm.DB
	blockchain *bcm.Blockchain
	genesisDoc *genesis.GenesisDoc
	logger     *logging.Logger
}

type ReplayCapture struct {
	Height        uint64
	AppHashBefore binary.HexBytes
	AppHashAfter  binary.HexBytes
	TxExecutions  []*exec.TxExecution
}

func (recap *ReplayCapture) String() string {
	return fmt.Sprintf("ReplayCapture[Height %d; AppHash: %v -> %v]",
		recap.Height, recap.AppHashBefore, recap.AppHashAfter)
}

func NewReplay(dbDir string, genesisDoc *genesis.GenesisDoc, logger *logging.Logger) *Replay {
	//burrowDB := core.NewBurrowDB(dbDir)
	// Avoid writing through to underlying DB
	db := dbm.NewDB(core.BurrowDBName, dbm.GoLevelDBBackend, dbDir)
	cacheDB := storage.NewCacheDB(db)
	return &Replay{
		explorer:   bcm.NewBlockExplorer(dbm.LevelDBBackend, path.Join(dbDir, "data")),
		db:         db,
		cacheDB:    cacheDB,
		blockchain: bcm.NewBlockchain(cacheDB, genesisDoc),
		genesisDoc: genesisDoc,
		logger:     logger,
	}
}

func (re *Replay) LatestBlockchain() (*bcm.Blockchain, error) {
	_, blockchain, err := bcm.LoadOrNewBlockchain(re.db, re.genesisDoc, re.logger)
	if err != nil {
		return nil, err
	}
	re.blockchain = blockchain
	return blockchain, nil
}

func (re *Replay) State(height uint64) (*state.State, error) {
	return state.LoadState(re.cacheDB, execution.VersionAtHeight(height))
}

func (re *Replay) Block(height uint64) (*ReplayCapture, error) {
	recap := new(ReplayCapture)
	// Load and commit previous block
	block, err := re.explorer.Block(int64(height - 1))
	if err != nil {
		return nil, err
	}
	err = re.blockchain.CommitBlockAtHeight(block.Time, block.Hash(), block.Header.AppHash, uint64(block.Height))
	if err != nil {
		return nil, err
	}
	// block.AppHash is hash after txs from previous block have been applied - it's the state we want to load on top
	// of which we will reapply this block txs
	st, err := re.State(height - 1)
	if err != nil {
		return nil, err
	}
	// Load block for replay
	block, err = re.explorer.Block(int64(height))
	if err != nil {
		return nil, err
	}
	if !bytes.Equal(st.Hash(), block.AppHash) {
		return nil, fmt.Errorf("state hash (%X) retrieved for block AppHash (%X) do not match",
			st.Hash(), block.AppHash)
	}
	recap.AppHashBefore = binary.HexBytes(block.AppHash)

	// Get our commit machinery
	committer := execution.NewBatchCommitter(st, execution.ParamsFromGenesis(re.genesisDoc), re.blockchain,
		event.NewEmitter(), re.logger)

	var txe *exec.TxExecution
	var execErr error
	_, err = block.Transactions(func(txEnv *txs.Envelope) (stop bool) {
		txe, execErr = committer.Execute(txEnv)
		if execErr != nil {
			return true
		}
		recap.TxExecutions = append(recap.TxExecutions, txe)
		return false
	})
	if err != nil {
		return nil, err
	}
	if execErr != nil {
		return nil, execErr
	}
	abciHeader := types.TM2PB.Header(&block.Header)
	recap.AppHashAfter, err = committer.Commit(&abciHeader)
	if err != nil {
		return nil, err
	}
	block, err = re.explorer.Block(int64(height + 1))
	if err != nil {
		return nil, err
	}
	fmt.Println(block.AppHash)
	return recap, nil
}

func (re *Replay) Blocks(startHeight, endHeight uint64) ([]*ReplayCapture, error) {
	var err error
	var st *state.State
	if startHeight > 1 {
		// Load and commit previous block
		block, err := re.explorer.Block(int64(startHeight - 1))
		if err != nil {
			return nil, err
		}
		err = re.blockchain.CommitBlockAtHeight(block.Time, block.Hash(), block.Header.AppHash, uint64(block.Height))
		if err != nil {
			return nil, err
		}
		// block.AppHash is hash after txs from previous block have been applied - it's the state we want to load on top
		// of which we will reapply this block txs
		st, err = re.State(startHeight - 1)
		if err != nil {
			return nil, err
		}
	} else {
		st, err = state.MakeGenesisState(re.cacheDB, re.genesisDoc)
		if err != nil {
			return nil, err
		}
		err = st.InitialCommit()
		if err != nil {
			return nil, err
		}
	}
	// Get our commit machinery
	committer := execution.NewBatchCommitter(st, execution.ParamsFromGenesis(re.genesisDoc), re.blockchain,
		event.NewEmitter(), re.logger)

	recaps := make([]*ReplayCapture, 0, endHeight-startHeight+1)
	for height := startHeight; height < endHeight; height++ {
		recap := &ReplayCapture{
			Height: height,
		}
		// Load block for replay
		block, err := re.explorer.Block(int64(height))
		if err != nil {
			return nil, errors.Wrap(err, "explorer.Block()")
		}
		if uint64(block.Height) != height {
			return nil, errors.Errorf("Tendermint block height %d != requested block height %d",
				block.Height, height)

		}
		if height > 1 && !bytes.Equal(st.Hash(), block.AppHash) {
			return nil, fmt.Errorf("state hash (%X) retrieved for block AppHash (%X) do not match",
				st.Hash(), block.AppHash[:])
		}
		recap.AppHashBefore = binary.HexBytes(block.AppHash)

		var txe *exec.TxExecution
		var execErr error
		_, err = block.Transactions(func(txEnv *txs.Envelope) (stop bool) {
			txe, execErr = committer.Execute(txEnv)
			if execErr != nil {
				return true
			}
			recap.TxExecutions = append(recap.TxExecutions, txe)
			return false
		})
		if err != nil {
			return nil, errors.Wrap(err, "block.Transactions()")
		}
		if execErr != nil {
			return nil, errors.Wrap(execErr, "committer.Execute()")
		}
		abciHeader := types.TM2PB.Header(&block.Header)
		recap.AppHashAfter, err = committer.Commit(&abciHeader)
		if err != nil {
			return nil, errors.Wrap(err, "committer.Commit()")
		}
		recaps = append(recaps, recap)
	}
	return recaps, nil
}
