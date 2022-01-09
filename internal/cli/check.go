package cli

import (
	"errors"
	"fmt"
	"os"

	"github.com/deref/transcript/internal/core"
	"github.com/sergi/go-diff/diffmatchpatch"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(checkCmd)
}

var checkCmd = &cobra.Command{
	Use:   "check <transcript>",
	Short: "Checks a transcript file",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		ckr := &core.Checker{}
		f, err := os.Open(args[0])
		if err != nil {
			return err
		}
		defer f.Close()
		err = ckr.CheckTranscript(ctx, f)
		var chkErr core.CommandCheckError
		if errors.As(err, &chkErr) {
			fmt.Printf("command on line %d failed check\n", chkErr.Lineno)
			fmt.Printf("$ %s\n", chkErr.Command)
			fmt.Println(chkErr.Err.Error())
			var diffErr core.DiffError
			if errors.As(err, &diffErr) {
				fmt.Println(diffmatchpatch.New().DiffPrettyText(diffErr.Diffs))
			}
		} else {
			return err
		}
		return nil
	},
}
