package cli

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	SilenceUsage:  true,
	SilenceErrors: true,
	Use:           "transcript",
	Args:          cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.SetArgs([]string{"shell"})
		return cmd.Execute()
	},
}

func Execute(ctx context.Context) {
	if err := rootCmd.ExecuteContext(ctx); err != nil {
		warnf("error: %v\n", err)
		os.Exit(1)
	}
}

func warnf(message string, v ...interface{}) {
	warn(fmt.Errorf(message, v...))
}

func warn(err error) {
	fmt.Fprintf(os.Stderr, "%v\n", err)
}
