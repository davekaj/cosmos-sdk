package main

import (
	"encoding/json"
	"os"

	"github.com/spf13/cobra"

	abci "github.com/tendermint/abci/types"
	"github.com/tendermint/tmlibs/cli"
	dbm "github.com/tendermint/tmlibs/db"
	"github.com/tendermint/tmlibs/log"

	"github.com/cosmos/cosmos-sdk/examples/ethermint/app"
	"github.com/cosmos/cosmos-sdk/server"
)

// ethermintdCmd is the entry point for this binary
var (
	context = server.NewDefaultContext()
	rootCmd = &cobra.Command{
		Use:               "ethermintd",
		Short:             "Ethermint Daemon (server)",
		PersistentPreRunE: server.PersistentPreRunEFn(context),
	}
)

func main() {
	cdc := app.MakeCodec()
	ctx := server.NewDefaultContext()

	server.AddCommands(ctx, cdc, rootCmd, server.DefaultAppInit,
		server.ConstructAppCreator(newApp, "ethermint"),
		server.ConstructAppExporter(exportAppState, "ethermint"))

	// prepare and add flags
	rootDir := os.ExpandEnv("$HOME/.ethermintd")
	executor := cli.PrepareBaseCmd(rootCmd, "EM", rootDir)
	executor.Execute()
}

func newApp(logger log.Logger, db dbm.DB) abci.Application {
	return app.NewEthermintApp(logger, db)
}

func exportAppState(logger log.Logger, db dbm.DB) (json.RawMessage, error) {
	bapp := app.NewEthermintApp(logger, db)
	return bapp.ExportAppStateJSON()
}
