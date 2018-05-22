package eth

import (
	"bytes"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	eth_types "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rlp"
)

// Wraps the RLP serialises representation of Ethereum transaction
type RawTxMsg struct {
	raw []byte
}

// enforce the msg type at compile time
var _ sdk.Msg = RawTxMsg{}

const EthRawMsgType = "ethraw"

// nolint
func (msg RawTxMsg) Type() string                            { return EthRawMsgType }
func (msg RawTxMsg) Get(key interface{}) (value interface{}) { return nil }
func (msg RawTxMsg) GetSigners() []sdk.Address               { return []sdk.Address{} }
func (msg RawTxMsg) String() string {
	return fmt.Sprintf("RawTxMsg{%x}", msg.raw)
}

// Validate Basic is used to quickly disqualify obviously invalid messages
func (msg RawTxMsg) ValidateBasic() sdk.Error {
	return nil
}

// Get the bytes for the message signer to sign on
func (msg RawTxMsg) GetSignBytes() []byte {
	return msg.raw
}

func (msg RawTxMsg) DecodeRaw() (*eth_types.Transaction, error) {
	tx := new(eth_types.Transaction)
	if err := tx.DecodeRLP(rlp.NewStream(bytes.NewReader(msg.raw), uint64(len(msg.raw)))); err != nil {
		return nil, err
	}
	return tx, nil
}
