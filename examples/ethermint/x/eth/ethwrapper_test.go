package eth

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	store "github.com/cosmos/cosmos-sdk/store"
	dbm "github.com/tendermint/tmlibs/db"
	"github.com/cosmos/cosmos-sdk/examples/ethermint/x/ethdb"

	eth_core "github.com/ethereum/go-ethereum/core"
	eth_vm "github.com/ethereum/go-ethereum/core/vm"
)

func TestInstantiate(t *testing.T) {
	var db dbm.DB = dbm.NewMemDB()
	multiStore := store.NewCommitMultiStore(db)
	storeKey := sdk.NewKVStoreKey("store1")
	multiStore.MountStoreWithDB(storeKey, sdk.StoreTypeIAVL, db)
	if err := multiStore.LoadLatestVersion(); err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	var kvStore sdk.KVStore = multiStore.GetKVStore(storeKey)
	kvd := ethdb.NewKVDatabase(kvStore)

	chainConfig, _, err := eth_core.SetupGenesisBlock(kvd, DefaultGenesis)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	_, err = eth_core.NewBlockChain(
		kvd,
		&eth_core.CacheConfig{},
		chainConfig,
		nil, // Consensus Engine
		eth_vm.Config{},
		)
	// Currently getting "Genesis not found on chain"
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
}