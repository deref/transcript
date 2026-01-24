package cli

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"runtime"
	"sync"
	"time"

	"github.com/deref/transcript/internal/core"
	"github.com/spf13/cobra"
)

func init() {
	checkCmd.Flags().IntVarP(&checkFlags.Jobs, "jobs", "j", 0, "maximum number of transcript files to check in parallel (0 = GOMAXPROCS)")
	checkCmd.Flags().BoolVarP(&checkFlags.Verbose, "verbose", "v", false, "verbose output")
	rootCmd.AddCommand(checkCmd)
}

var checkFlags struct {
	Jobs    int
	Verbose bool
}

var checkCmd = &cobra.Command{
	Use:   "check <transcripts...>",
	Short: "Checks transcript files",
	Long: `Checks transcript files.

When multiple transcripts are provided, checks run in parallel by default.
Use -j 1 to force sequential checking if your transcripts share mutable
external state.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			warnf("no transcripts to check")
			os.Exit(1)
		}
		failures, err := runCheck(cmd.Context(), checkOptions{
			Filenames: args,
			Out:       cmd.OutOrStdout(),
			Jobs:      checkFlags.Jobs,
			Verbose:   checkFlags.Verbose,
		})
		if err != nil {
			return err
		}
		if failures > 0 {
			os.Exit(1)
		}
		return nil
	},
}

type checkOptions struct {
	Filenames []string
	Out       io.Writer
	Jobs      int
	Verbose   bool
}

func runCheck(ctx context.Context, opts checkOptions) (failures int, err error) {
	filenames := opts.Filenames
	out := opts.Out
	jobs := opts.Jobs
	verbose := opts.Verbose

	if jobs <= 0 {
		jobs = runtime.GOMAXPROCS(0)
	}
	if jobs < 1 {
		jobs = 1
	}
	if jobs > len(filenames) {
		jobs = len(filenames)
	}

	if verbose {
		for _, filename := range filenames {
			fmt.Fprintf(out, "=== RUN   %s\n", filename)
		}
	}

	type task struct {
		idx      int
		filename string
	}
	type result struct {
		idx    int
		ok     bool
		output string
		dur    time.Duration
		err    error
	}

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	tasks := make(chan task)
	results := make(chan result, jobs)

	var wg sync.WaitGroup
	wg.Add(jobs)
	for i := 0; i < jobs; i++ {
		go func() {
			defer wg.Done()
			for {
				select {
				case <-ctx.Done():
					return
				case t, ok := <-tasks:
					if !ok {
						return
					}
					ok, output, dur, err := checkFile(ctx, t.filename)
					if err != nil {
						cancel()
					}
					results <- result{idx: t.idx, ok: ok, output: output, dur: dur, err: err}
				}
			}
		}()
	}
	go func() {
		wg.Wait()
		close(results)
	}()

	go func() {
		defer close(tasks)
		for i, filename := range filenames {
			select {
			case <-ctx.Done():
				return
			case tasks <- task{idx: i, filename: filename}:
			}
		}
	}()

	var firstErr error
	byIndex := make([]*result, len(filenames))
	nextToPrint := 0
	for r := range results {
		r := r // copy for pointer stability
		byIndex[r.idx] = &r
		if firstErr == nil && r.err != nil {
			firstErr = r.err
		}
		for nextToPrint < len(byIndex) && byIndex[nextToPrint] != nil {
			res := byIndex[nextToPrint]
			if res.output != "" {
				fmt.Fprint(out, res.output)
			}
			if verbose {
				status := "PASS"
				if !res.ok {
					status = "FAIL"
				}
				fmt.Fprintf(out, "--- %s: %s (%.2fs)\n", status, filenames[nextToPrint], res.dur.Seconds())
			}
			if !res.ok {
				failures++
			}
			nextToPrint++
		}
	}
	return failures, firstErr
}

func checkFile(ctx context.Context, filename string) (ok bool, output string, dur time.Duration, err error) {
	start := time.Now()
	var buf bytes.Buffer
	ok, err = checkFileToWriter(ctx, filename, &buf)
	return ok, buf.String(), time.Since(start), err
}

func checkFileToWriter(ctx context.Context, filename string, out io.Writer) (ok bool, err error) {
	f, err := os.Open(filename)
	if err != nil {
		return false, err
	}
	defer f.Close()

	ckr := &core.Checker{}
	err = ckr.CheckTranscript(ctx, f)
	var chkErr core.CommandCheckError
	if errors.As(err, &chkErr) {
		fmt.Fprintf(out, "failed check at %s:%d\n", filename, chkErr.Lineno)
		fmt.Fprintf(out, "$ %s\n", chkErr.Command)
		for _, err := range chkErr.Errs {
			fmt.Fprintln(out, err.Error())
			var diffErr core.DiffError
			if errors.As(err, &diffErr) {
				if color {
					fmt.Fprint(out, diffErr.Color())
				} else {
					fmt.Fprint(out, diffErr.Plain())
				}
			}
		}
		return false, nil
	}
	return err == nil, err
}
