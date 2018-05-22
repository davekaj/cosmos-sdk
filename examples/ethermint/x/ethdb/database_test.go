package ethdb

import (
	"bytes"
	"testing"
	
	sdk "github.com/cosmos/cosmos-sdk/types"
	store "github.com/cosmos/cosmos-sdk/store"
	dbm "github.com/tendermint/tmlibs/db"
)

func TestBatchCreation(t *testing.T) {
	var db dbm.DB = dbm.NewMemDB()
	multiStore := store.NewCommitMultiStore(db)
	storeKey := sdk.NewKVStoreKey("store1")
	multiStore.MountStoreWithDB(storeKey, sdk.StoreTypeIAVL, db)
	if err := multiStore.LoadLatestVersion(); err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	var kvStore sdk.KVStore = multiStore.GetKVStore(storeKey)
	kvd := NewKVDatabase(kvStore)
	kvb := kvd.NewBatch()
	if err := kvb.Put([]byte("key"), []byte("value")); err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if err := kvb.Write(); err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	v, err := kvd.Get([]byte("key"))
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if !bytes.Equal(v, []byte("value")) {
		t.Errorf("Expected to write the key through")
	}
}
