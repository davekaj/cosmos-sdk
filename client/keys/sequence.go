package keys

import (
	"fmt"

	"github.com/cosmos/cosmos-sdk/client/sequence"
	"github.com/spf13/cobra"
)

var localSequenceCommand = &cobra.Command{
	Use:   "localSequence <name>",
	Short: "Show sequence info for the given name",
	Long:  `Return sequence info for local key.`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		name := args[0]
		seq, err := sequence.GetLocalSequence(name)
		if err != nil {
			fmt.Println(err)
			return
		}
		fmt.Println("name: ", name, " sequence: ", seq)
		return
	},
}
