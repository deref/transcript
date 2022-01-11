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
	updateCmd.Flags().BoolVarP(&updateFlags.DryRun, "dry-run", "n", false, "dry run")
	rootCmd.AddCommand(updateCmd)
}

var updateFlags struct {
	DryRun bool
}

var updateCmd = &cobra.Command{
	Use:   "update <transcripts...>",
	Short: "Updates transcript files",
	Long: `Updates output and exit code expectations in transcript files.
	
Transcript files are updated in-place, unless --dry-run is specified. In a dry
run, the updated output is printed to stdout instead.
`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		for _, filename := range args {
			if err := updateFile(ctx, filename); err != nil {
				return fmt.Errorf("updating %q: %w", filename, err)
			}
		}
		return nil
	},
}

func updateFile(ctx context.Context, filename string) error {
	f, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer f.Close()

	upr := &core.Updater{}
	transcript, err := upr.UpdateTranscript(ctx, f)
	if err != nil {
		return err
	}
	if updateFlags.DryRun {
		_, err := io.Copy(os.Stdout, transcript)
		return err
	}
	return atomic.WriteFile(filename, transcript)
}
