package sequence

import (
	"encoding/binary"
	"fmt"
	"path/filepath"

	"github.com/spf13/viper"
	"github.com/tendermint/tmlibs/cli"
	dbm "github.com/tendermint/tmlibs/db"
)

//A simple db to store account sequences. Used mainly for keeping track of the sequence when sending multiple async functions

const SequenceDBName = "sequence"

var _ dbm.DB = (*dbm.GoLevelDB)(nil)
var sequenceStore dbm.GoLevelDB

// initialize a keybase based on the configuration
func GetSequenceStore() (dbm.DB, error) {
	rootDir := viper.GetString(cli.HomeFlag)
	return GetSequenceStoreFromDir(rootDir)
}

// initialize a keybase based on the configuration
func GetSequenceStoreFromDir(rootDir string) (dbm.DB, error) {
	db, err := dbm.NewGoLevelDB(SequenceDBName, filepath.Join(rootDir, "sequence"))
	if err != nil {
		return nil, err
	}
	return db, nil
}

func GetLocalSequence(account string) (int64, error) {
	ss, err := GetSequenceStore()
	defer ss.Close()
	if err != nil {
		return 0, err
	}
	sequenceBytes := ss.Get(stringToByteArray(account))
	if err != nil {
		return 0, err
	}
	sequenceInt := bytes2Int64(sequenceBytes)
	return sequenceInt, nil

}

func SetLocalSequence(account string, newSequence int64) {
	ss, err := GetSequenceStore()
	defer ss.Close()
	if err != nil {
		fmt.Println("errorB: ", err)
	}
	ss.Set(stringToByteArray(account), int642Bytes(newSequence))
}

func HasLocalSequence(account string) (bool, error) {
	ss, err := GetSequenceStore()
	defer ss.Close()
	if err != nil {
		return false, err
	}
	isStored := ss.Has(stringToByteArray(account))
	if err != nil {
		return false, err
	}
	return isStored, nil
}

////////////////////////////////////////////////////////////////////Helpers

func stringToByteArray(name string) []byte {
	return []byte(fmt.Sprintf("%s.info", name))
}

func int642Bytes(i int64) []byte {
	buf := make([]byte, 8)
	binary.BigEndian.PutUint64(buf, uint64(i))
	return buf
}

func bytes2Int64(buf []byte) int64 {
	return int64(binary.BigEndian.Uint64(buf))
}
