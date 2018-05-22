package eth

import (
	"bytes"
	"encoding/hex"
	"math/big"
	"testing"
)

func TestDecodeRaw(t *testing.T) {
	// This is transaction with hash 0xbe4bd5f9f8af8c7571f876cf7dd45e6d97edb659b2295e475e49dd6a83486064 from block 2702566 on Ethereum mainnet
	rawTxHex := "f8ac820bf785060db884008304baf094a43ebd8939d8328f5858119a3fb65f65c864c6dd80b84453" +
		"f11cb39ff770adc1ec21ba84b44feba7167e1677d2d1d35cd36f763d97c406a6e907fd00000000000000000000000000000" +
		"000000000000000000000000000000001401ca053ef121cc02a9f0837fcd662f4c5c34f59d8e995d6609a17ad9348481bfd" +
		"966ba04304ae6c032a716de0182510fd1fcc7644a197483e30d33a6e591cb8b0431f52"
	rawtx := RawTxMsg{}
	var err error
	rawtx.raw = make([]byte, len(rawTxHex)/2)
	_, err = hex.Decode(rawtx.raw, []byte(rawTxHex))
	if err != nil {
		t.Errorf("Could not decode raw tx string: %v", err)
	}
	ethTx, err := rawtx.DecodeRaw()
	if err != nil {
		t.Errorf("Could not decode raw Eth tx: %v", err)
	}
	chainId := ethTx.ChainId()
	if chainId.Cmp(big.NewInt(0)) != 0 {
		t.Errorf("Wrong chain id, got %d, want %d", chainId, 0)
	}
	hash := ethTx.Hash()
	wantHashStr := "be4bd5f9f8af8c7571f876cf7dd45e6d97edb659b2295e475e49dd6a83486064"
	wantHash := make([]byte, len(wantHashStr)/2)
	hex.Decode(wantHash, []byte(wantHashStr))
	if !bytes.Equal(hash[:], wantHash[:]) {
		t.Errorf("Wrong tx hash, got %x, want %x", hash, wantHash)
	}
}
