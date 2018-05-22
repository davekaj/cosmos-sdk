package eth

import (
	"math/big"

	eth_common "github.com/ethereum/go-ethereum/common"
	eth_core "github.com/ethereum/go-ethereum/core"
	eth_params "github.com/ethereum/go-ethereum/params"
	//eth_types "github.com/ethereum/go-ethereum/core/types"
)

var DefaultChainConfig = &eth_params.ChainConfig{
	ChainId:             big.NewInt(1),
	HomesteadBlock:      big.NewInt(1),
	DAOForkBlock:        big.NewInt(1),
	DAOForkSupport:      false,
	EIP150Block:         big.NewInt(1),
	EIP150Hash:          eth_common.Hash{}, // Optional feature
	EIP155Block:         big.NewInt(1),
	EIP158Block:         big.NewInt(1),
	ByzantiumBlock:      big.NewInt(1),
	ConstantinopleBlock: nil,
	Ethash:              nil,
}

var DefaultGenesis = &eth_core.Genesis{
	Config: DefaultChainConfig,
}

// Instantiation of the ethereum state processor based on KVStore and tendermint consensus
type StateProcessor struct {
	blockchain *eth_core.BlockChain
}

func NewStateProcessor() {

}
