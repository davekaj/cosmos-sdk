package cli

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/cosmos/cosmos-sdk/client/context"
	"github.com/cosmos/cosmos-sdk/client/sequence"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/wire"
	authcmd "github.com/cosmos/cosmos-sdk/x/auth/client/cli"
	"github.com/cosmos/cosmos-sdk/x/bank/client"
)

const (
	flagTo     = "to"
	flagAmount = "amount"
	flagAsync  = "async"
)

// SendTxCommand will create a send tx and sign it with the given key
func SendTxCmd(cdc *wire.Codec) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "send",
		Short: "Create and sign a send tx",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.NewCoreContextFromViper().WithDecoder(authcmd.GetAccountDecoder(cdc))

			// get the from/to address
			from, err := ctx.GetFromAddress()
			if err != nil {
				return err
			}

			toStr := viper.GetString(flagTo)

			to, err := sdk.GetAccAddressBech32(toStr)
			if err != nil {
				return err
			}
			// parse coins
			amount := viper.GetString(flagAmount)
			coins, err := sdk.ParseCoins(amount)
			if err != nil {
				return err
			}

			// build and sign the transaction, then broadcast to Tendermint
			msg := client.BuildMsg(from, to, coins)

			asyncStr := viper.GetString(flagAsync)

			if asyncStr == "true" {
				res, err := ctx.SignBuildBroadcastAsync(ctx.FromAddressName, msg, cdc)
				if err != nil {
					return err
				}
				localSeq, _ := sequence.GetLocalSequence(ctx.FromAddressName)
				sequence.SetLocalSequence(ctx.FromAddressName, localSeq+1)
				fmt.Println("Async tx send. tx hash: ", res.Hash)
				return nil
			} else {
				res, err := ctx.EnsureSignBuildBroadcast(ctx.FromAddressName, msg, cdc)
				if err != nil {
					// Check to see if the local sequence isn't in sync with the sequence stored on tendermint
					// If it isn't in sync, update them to match, and try the Tx again
					// otherwise, return the error
					localSeq, _ := sequence.GetLocalSequence(ctx.FromAddressName)
					from, err := ctx.GetFromAddress()
					if err != nil {
						return err
					}
					seq, err := ctx.NextSequence(from)
					if err != nil {
						return err
					}
					if localSeq != seq {
						fmt.Printf("The local account sequence did not match the sequence stored on the blockchain. Updated local sequence to : %d\n", seq)
						fmt.Println("Attempting to send tx with updated sequence")
						sequence.SetLocalSequence(ctx.FromAddressName, seq)
						res, err = ctx.EnsureSignBuildBroadcast(ctx.FromAddressName, msg, cdc)
						if err != nil {
							return err
						}
						return nil
					}
					return err
				}
				localSeq, _ := sequence.GetLocalSequence(ctx.FromAddressName)
				sequence.SetLocalSequence(ctx.FromAddressName, localSeq+1)
				fmt.Printf("Committed at block %d. Hash: %s\n", res.Height, res.Hash.String())
				return nil
			}
		},
	}

	cmd.Flags().String(flagTo, "", "Address to send coins")
	cmd.Flags().String(flagAmount, "", "Amount of coins to send")
	cmd.Flags().String(flagAsync, "", "Pass the async flag to send a tx without waiting for the tx to be included in a block")
	return cmd
}
