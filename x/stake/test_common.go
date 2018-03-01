package stake

import (
	"encoding/hex"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	crypto "github.com/tendermint/go-crypto"
	dbm "github.com/tendermint/tmlibs/db"

	"github.com/cosmos/cosmos-sdk/store"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func subspace(prefix []byte) (start, end []byte) {
	end = make([]byte, len(prefix))
	copy(end, prefix)
	end[len(end)-1]++
	return prefix, end
}

func initTestStore(t *testing.T) sdk.KVStore {
	// Capabilities key to access the main KVStore.
	//db, err := dbm.NewGoLevelDB("stake", "data")
	db := dbm.NewMemDB()
	stakeStoreKey := sdk.NewKVStoreKey("stake")
	ms := store.NewCommitMultiStore(db)
	ms.MountStoreWithDB(stakeStoreKey, sdk.StoreTypeIAVL, db)
	err := ms.LoadLatestVersion()
	require.Nil(t, err)
	return ms.GetKVStore(stakeStoreKey)
}

func newAddrs(n int) (addrs []crypto.Address) {
	for i := 0; i < n; i++ {
		addrs = append(addrs, []byte(fmt.Sprintf("addr%d", i)))
	}
	return
}

func newPubKey(pk string) (res crypto.PubKey) {
	pkBytes, err := hex.DecodeString(pk)
	if err != nil {
		panic(err)
	}
	//res, err = crypto.PubKeyFromBytes(pkBytes)
	var pkEd crypto.PubKeyEd25519
	copy(pkEd[:], pkBytes[:])
	return pkEd
}

// dummy pubkeys used for testing
var pks = []crypto.PubKey{
	newPubKey("0B485CFC0EECC619440448436F8FC9DF40566F2369E72400281454CB552AFB51"),
	newPubKey("0B485CFC0EECC619440448436F8FC9DF40566F2369E72400281454CB552AFB52"),
	newPubKey("0B485CFC0EECC619440448436F8FC9DF40566F2369E72400281454CB552AFB53"),
	newPubKey("0B485CFC0EECC619440448436F8FC9DF40566F2369E72400281454CB552AFB54"),
	newPubKey("0B485CFC0EECC619440448436F8FC9DF40566F2369E72400281454CB552AFB55"),
	newPubKey("0B485CFC0EECC619440448436F8FC9DF40566F2369E72400281454CB552AFB56"),
	newPubKey("0B485CFC0EECC619440448436F8FC9DF40566F2369E72400281454CB552AFB57"),
	newPubKey("0B485CFC0EECC619440448436F8FC9DF40566F2369E72400281454CB552AFB58"),
	newPubKey("0B485CFC0EECC619440448436F8FC9DF40566F2369E72400281454CB552AFB59"),
	newPubKey("0B485CFC0EECC619440448436F8FC9DF40566F2369E72400281454CB552AFB60"),
}

// NOTE: PubKey is supposed to be the binaryBytes of the crypto.PubKey
// instead this is just being set the address here for testing purposes
func candidatesFromActors(store sdk.KVStore, addrs []crypto.Address, amts []int64) {
	for i := 0; i < len(addrs); i++ {
		c := &Candidate{
			Status:      Unbonded,
			PubKey:      pks[i],
			Owner:       addrs[i],
			Assets:      sdk.NewRat(amts[i]),
			Liabilities: sdk.NewRat(amts[i]),
			VotingPower: sdk.NewRat(amts[i]),
		}
		saveCandidate(store, c)
	}
}

func candidatesFromActorsEmpty(addrs []crypto.Address) (candidates Candidates) {
	for i := 0; i < len(addrs); i++ {
		c := &Candidate{
			Status:      Unbonded,
			PubKey:      pks[i],
			Owner:       addrs[i],
			Assets:      sdk.ZeroRat,
			Liabilities: sdk.ZeroRat,
			VotingPower: sdk.ZeroRat,
		}
		candidates = append(candidates, c)
	}
	return
}

//// helper function test if Candidate is changed asabci.Validator
//func testChange(t *testing.T, val Validator, chg *abci.Validator) {
//assert := assert.New(t)
//assert.Equal(val.PubKey.Bytes(), chg.PubKey)
//assert.Equal(val.VotingPower.Evaluate(), chg.Power)
//}

//// helper function test if Candidate is removed as abci.Validator
//func testRemove(t *testing.T, val Validator, chg *abci.Validator) {
//assert := assert.New(t)
//assert.Equal(val.PubKey.Bytes(), chg.PubKey)
//assert.Equal(int64(0), chg.Power)
//}