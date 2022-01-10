package interactive

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/chzyer/readline"
	"github.com/deref/transcript/internal/core"
)

type Shell struct {
	rec *core.Recorder
	rl  *readline.Instance
}

func (sh *Shell) Run(ctx context.Context) error {
	var err error
	sh.rl, err = readline.New("$ ")
	if err != nil {
		return fmt.Errorf("initializing readline: %w", err)
	}
	defer sh.rl.Close()

	sh.rec = &core.Recorder{
		Stdout: os.Stdout,
		Stderr: os.Stderr,
	}
	if err := sh.rec.Init(); err != nil {
		return fmt.Errorf("initializing: %w", err)
	}

	for {
		line, err := sh.rl.Readline()
		if errors.Is(err, io.EOF) {
			return nil
		}
		if err != nil {
			return fmt.Errorf("readline: %w", err)
		}
		res, err := sh.rec.RunCommand(ctx, line)
		if err != nil {
			return err
		}
		if res.ExitCode != 0 {
			fmt.Fprintf(os.Stderr, "? %d\n", res.ExitCode)
		}
		if sh.rec.Exited() {
			return nil
		}
	}
}

func (sh *Shell) DumpTranscript(w io.Writer) error {
	_, err := io.Copy(w, &sh.rec.Transcript)
	return err
}
