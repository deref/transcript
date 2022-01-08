package cli

import (
	"fmt"
	"io/ioutil"

	"github.com/deref/transcript/internal/interactive"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(shellCmd)
}

var shellCmd = &cobra.Command{
	Use:   "shell",
	Short: "Runs an interactive subshell",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		sh := &interactive.Shell{}
		if err := sh.Run(ctx); err != nil {
			return err
		}

		file, err := ioutil.TempFile("", "transcript")
		if err != nil {
			return fmt.Errorf("creating tempfile: %w", err)
		}
		if err := sh.DumpTranscript(file); err != nil {
			return fmt.Errorf("writing tempfile: %w", err)
		}
		_, _ = fmt.Printf("wrote transcript: %s\n", file.Name())
		return nil
	},
}
