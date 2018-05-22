package app

import (
	"encoding/json"
	"fmt"

	abci "github.com/tendermint/abci/types"
	cmn "github.com/tendermint/tmlibs/common"
	dbm "github.com/tendermint/tmlibs/db"
	"github.com/tendermint/tmlibs/log"

	bam "github.com/cosmos/cosmos-sdk/baseapp"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/wire"
	"github.com/cosmos/cosmos-sdk/x/auth"
	"github.com/cosmos/cosmos-sdk/x/bank"
	"github.com/cosmos/cosmos-sdk/x/ibc"
	"github.com/cosmos/cosmos-sdk/x/stake"

	"github.com/cosmos/cosmos-sdk/examples/ethermint/types"
	"github.com/cosmos/cosmos-sdk/examples/ethermint/x/eth"

	"github.com/ethereum/go-ethereum/rlp"
)

const (
	appName = "Ethermint"
)

var k = rlp.String

// Extended ABCI application
type EthermintApp struct {
	*bam.BaseApp
	cdc *wire.Codec

	// keys to access the substores
	keyMain    *sdk.KVStoreKey
	keyAccount *sdk.KVStoreKey
	keyIBC     *sdk.KVStoreKey
	keyStake   *sdk.KVStoreKey

	// Manage getting and setting accounts
	accountMapper sdk.AccountMapper

	// Handle fees
	feeHandler sdk.FeeHandler
}

func NewEthermintApp(logger log.Logger, db dbm.DB) *EthermintApp {
	// Create app-level codec for txs and accounts.
	var cdc = MakeCodec()

	// create your application object
	var app = &EthermintApp{
		BaseApp:            bam.NewBaseApp(appName, cdc, logger, db),
		cdc:                cdc,
		keyMain:    sdk.NewKVStoreKey("main"),
		keyAccount: sdk.NewKVStoreKey("acc"),
		keyIBC:     sdk.NewKVStoreKey("ibc"),
		keyStake:   sdk.NewKVStoreKey("staking"),
	}

	// define the accountMapper
	app.accountMapper = auth.NewAccountMapper(
		cdc,
		app.keyAccount, // target store
		&types.AppAccount{}, // prototype
	)

	// add handlers
	coinKeeper := bank.NewKeeper(app.accountMapper)
	ibcMapper := ibc.NewMapper(app.cdc, app.keyIBC, app.RegisterCodespace(ibc.DefaultCodespace))
	stakeKeeper := stake.NewKeeper(app.cdc, app.keyStake, coinKeeper, app.RegisterCodespace(stake.DefaultCodespace))
	app.Router().
		AddRoute("bank", bank.NewHandler(coinKeeper)).
		AddRoute("ibc", ibc.NewHandler(ibcMapper, coinKeeper)).
		AddRoute("simplestake", stake.NewHandler(stakeKeeper))

	// Define the feeHandler.
	app.feeHandler = auth.BurnFeeHandler

	// initialize BaseApp
	app.SetInitChainer(app.initChainer)
	app.MountStoresIAVL(app.keyMain, app.keyAccount, app.keyIBC, app.keyStake)
	app.SetAnteHandler(auth.NewAnteHandler(app.accountMapper, app.feeHandler))
	err := app.LoadLatestVersion(app.keyMain)
	if err != nil {
		cmn.Exit(err.Error())
	}

	return app
}

// custom tx codec
// TODO: use new go-wire
func MakeCodec() *wire.Codec {
	var cdc = wire.NewCodec()

	// Register Msgs
	wire.RegisterCrypto(cdc)
	sdk.RegisterWire(cdc)
	bank.RegisterWire(cdc)
	stake.RegisterWire(cdc)
	ibc.RegisterWire(cdc)

	cdc.RegisterConcrete(eth.RawTxMsg{}, "ethermint/RawEth", nil)

		// Register AppAccount
	cdc.RegisterInterface((*sdk.Account)(nil), nil)
	cdc.RegisterConcrete(&types.AppAccount{}, "ethermint/Account", nil)

	return cdc
}

// custom logic for transaction decoding
func (app *EthermintApp) txDecoder(txBytes []byte) (sdk.Tx, sdk.Error) {
	var tx = sdk.StdTx{}

	if len(txBytes) == 0 {
		return nil, sdk.ErrTxDecode("txBytes are empty")
	}

	// StdTx.Msg is an interface. The concrete types
	// are registered by MakeTxCodec in bank.RegisterWire.
	err := app.cdc.UnmarshalBinary(txBytes, &tx)
	if err != nil {
		return nil, sdk.ErrTxDecode("").Trace(err.Error())
	}
	if tx.GetMsg().Type() == eth.EthRawMsgType {
		if ethrawtx, ok := tx.GetMsg().(eth.RawTxMsg); ok {
			ethrawtx.DecodeRaw()
		} else {
			return nil, sdk.ErrTxDecode(fmt.Sprintf("tx is interpreted as EthRaw, but cannot be cast to apppropriate type: %x", ethrawtx))
		}
	}
	return tx, nil
}

// custom logic for basecoin initialization
func (app *EthermintApp) initChainer(ctx sdk.Context, req abci.RequestInitChain) abci.ResponseInitChain {
	stateJSON := req.AppStateBytes

	genesisState := new(types.GenesisState)
	err := json.Unmarshal(stateJSON, genesisState)
	if err != nil {
		panic(err) // TODO https://github.com/cosmos/cosmos-sdk/issues/468
		// return sdk.ErrGenesisParse("").TraceCause(err, "")
	}

	for _, gacc := range genesisState.Accounts {
		acc, err := gacc.ToAppAccount()
		if err != nil {
			panic(err) // TODO https://github.com/cosmos/cosmos-sdk/issues/468
			//	return sdk.ErrGenesisParse("").TraceCause(err, "")
		}
		app.accountMapper.SetAccount(ctx, acc)
	}
	return abci.ResponseInitChain{}
}

// Custom logic for state export
func (app *EthermintApp) ExportAppStateJSON() (appState json.RawMessage, err error) {
	ctx := app.NewContext(true, abci.Header{})

	// iterate to get the accounts
	accounts := []*types.GenesisAccount{}
	appendAccount := func(acc sdk.Account) (stop bool) {
		account := &types.GenesisAccount{
			Address: acc.GetAddress(),
			Coins:   acc.GetCoins(),
		}
		accounts = append(accounts, account)
		return false
	}
	app.accountMapper.IterateAccounts(ctx, appendAccount)

	genState := types.GenesisState{
		Accounts: accounts,
	}
	return wire.MarshalJSONIndent(app.cdc, genState)
}
