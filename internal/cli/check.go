package cli

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/deref/transcript/internal/core"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(checkCmd)
}

var checkCmd = &cobra.Command{
	Use:   "check <transcripts...>",
	Short: "Checks transcript files",
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			warnf("no transcripts to check")
			os.Exit(1)
		}
		ctx := cmd.Context()
		failures := 0
		for _, filename := range args {
			ok, err := checkFile(ctx, filename)
			if !ok {
				failures++
			}
			if err != nil {
				return err
			}
		}
		if failures > 0 {
			os.Exit(1)
		}
		return nil
	},
}

func checkFile(ctx context.Context, filename string) (ok bool, err error) {
	f, err := os.Open(filename)
	if err != nil {
		return false, err
	}
	defer f.Close()

	ckr := &core.Checker{}
	err = ckr.CheckTranscript(ctx, f)
	var chkErr core.CommandCheckError
	if errors.As(err, &chkErr) {
		fmt.Printf("failed check at %s:%d\n", filename, chkErr.Lineno)
		fmt.Printf("$ %s\n", chkErr.Command)
		for _, err := range chkErr.Errs {
			fmt.Println(err.Error())
			var diffErr core.DiffError
			if errors.As(err, &diffErr) {
				if color {
					fmt.Print(diffErr.Color())
				} else {
					fmt.Print(diffErr.Plain())
				}
			}
		}
		return false, nil
	}
	return err == nil, err
}
