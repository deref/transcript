package cli

import (
	"fmt"
	"os"

	"github.com/deref/transcript/internal/interactive"
	"github.com/spf13/cobra"
)

func init() {
	shellCmd.Flags().StringVarP(&shellFlags.OutputPath, "output", "o", "", "output file path")
	rootCmd.AddCommand(shellCmd)
}

var shellFlags struct {
	OutputPath string
}

var shellCmd = &cobra.Command{
	Use:   "shell",
	Short: "Runs an interactive subshell",
	Long: `Runs an interactive subshell and writes a transcript of the session to
a file.

If --output is not specified, a tempfile will be written.
`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		sh := &interactive.Shell{}
		if err := sh.Run(ctx); err != nil {
			return err
		}

		var file *os.File
		var err error
		if shellFlags.OutputPath == "" {
			file, err = os.CreateTemp("", "transcript")
		} else {
			file, err = os.OpenFile(shellFlags.OutputPath, os.O_CREATE|os.O_TRUNC|os.O_RDWR, 0600)
		}
		if err != nil {
			return fmt.Errorf("creating output: %w", err)
		}
		defer file.Close()

		if err := sh.DumpTranscript(file); err != nil {
			return fmt.Errorf("writing output: %w", err)
		}
		if shellFlags.OutputPath == "" {
			_, _ = fmt.Printf("wrote transcript: %s\n", file.Name())
		}
		return nil
	},
}
