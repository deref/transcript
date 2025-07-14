package cli

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/deref/transcript/internal/core"
	"github.com/natefinch/atomic"
	"github.com/spf13/cobra"
)

func init() {
	formatCmd.Flags().BoolVarP(&formatFlags.DryRun, "dry-run", "n", false, "dry run")
	rootCmd.AddCommand(formatCmd)
}

var formatFlags struct {
	DryRun bool
}

var formatCmd = &cobra.Command{
	Use:   "format <transcripts...>",
	Short: "Formats transcript files",
	Long: `Formats transcript files by normalizing comments, blank lines,
trailing whitespace (except in command output), trailing newline,
and special directive syntax.

Transcript files are formatted in-place, unless --dry-run is specified. In a dry
run, the formatted output is printed to stdout instead.
`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		for _, filename := range args {
			if err := formatFile(ctx, filename); err != nil {
				return fmt.Errorf("formatting %q: %w", filename, err)
			}
		}
		return nil
	},
}

func formatFile(ctx context.Context, filename string) error {
	f, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer f.Close()

	formatter := &core.Formatter{}
	transcript, err := formatter.FormatTranscript(ctx, f)
	if err != nil {
		return err
	}
	if formatFlags.DryRun {
		_, err := io.Copy(os.Stdout, transcript)
		return err
	}
	return atomic.WriteFile(filename, transcript)
}