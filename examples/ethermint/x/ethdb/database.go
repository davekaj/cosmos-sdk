package ethdb

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	eth_ethdb "github.com/ethereum/go-ethereum/ethdb"
)

type KVDatabase struct {
	store sdk.KVStore
}

func NewKVDatabase(store sdk.KVStore) *KVDatabase {
	return &KVDatabase{store: store}
}

// Implementing ethdb.Database from go-ethereum
func (kvd *KVDatabase) Put(key []byte, value []byte) error {
	kvd.store.Set(key, value)
	return nil
}

func (kvd *KVDatabase) Get(key []byte) ([]byte, error) {
	return kvd.store.Get(key), nil
}

func (kvd *KVDatabase) Has(key []byte) (bool, error) {
	return kvd.store.Has(key), nil
}

func (kvd *KVDatabase) Delete(key []byte) error {
	kvd.store.Delete(key)
	return nil
}

func (kvd *KVDatabase) Close() {
}

type KVBatch struct {
	underlying sdk.KVStore // Kept only to be able to reset
	store sdk.CacheKVStore
}

func (kvd *KVDatabase) NewBatch() eth_ethdb.Batch {
	return &KVBatch{underlying: kvd.store, store: kvd.store.CacheWrap().(sdk.CacheKVStore)}
}

func (kvb *KVBatch) Put(key []byte, value []byte) error {
	kvb.store.Set(key, value)
	return nil
}

func (kvb *KVBatch) ValueSize() int {
	// Has to return an arbitrary value because there is nowhere to take it from
	return 0
}

func (kvb *KVBatch) Write() error {
	kvb.store.Write()
	return nil
}

func (kvb *KVBatch) Reset() {
	kvb.store = kvb.underlying.CacheWrap().(sdk.CacheKVStore)
}
