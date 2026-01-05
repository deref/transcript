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
	rootCmd.AddCommand(formatCmd)
}

var formatCmd = &cobra.Command{
	Use:   "format [transcripts...]",
	Short: "Formats transcript files",
	Long: `Formats transcript files by normalizing comments, blank lines,
trailing whitespace (except in command output), trailing newline,
and special directive syntax.

If no files are provided, reads from stdin and writes to stdout.
If files are provided, formats them in-place.
`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		if len(args) == 0 {
			// Read from stdin, write to stdout
			return formatStdin(ctx)
		}
		// Format files in-place
		for _, filename := range args {
			if err := formatFile(ctx, filename); err != nil {
				return fmt.Errorf("formatting %q: %w", filename, err)
			}
		}
		return nil
	},
}

func formatStdin(ctx context.Context) error {
	formatter := &core.Formatter{}
	transcript, err := formatter.FormatTranscript(ctx, os.Stdin)
	if err != nil {
		return err
	}
	_, err = io.Copy(os.Stdout, transcript)
	return err
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
	return atomic.WriteFile(filename, transcript)
}
