package core

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"mvdan.cc/sh/v3/interp"
	"mvdan.cc/sh/v3/syntax"
)

type Recorder struct {
	Stdout *os.File
	Stderr *os.File

	Transcript bytes.Buffer

	runner *interp.Runner
	stdout *lineBufferingWriter
	stderr *lineBufferingWriter
}

func (rec *Recorder) Init() error {
	var err error
	rec.stdout = &lineBufferingWriter{
		W: &rec.Transcript,
	}
	rec.stderr = &lineBufferingWriter{
		W: &rec.Transcript,
	}
	rec.runner, err = interp.New(
		interp.StdIO(nil,
			io.MultiWriter(rec.Stdout, &prefixingWriter{
				Prefix: "1 ",
				W:      rec.stdout,
			}),
			io.MultiWriter(rec.Stderr, &prefixingWriter{
				Prefix: "2 ",
				W:      rec.stderr,
			}),
		))
	return err
}

func (rec *Recorder) flush() error {
	if err := rec.stdout.Flush(); err != nil {
		return err
	}
	if err := rec.stderr.Flush(); err != nil {
		return err
	}
	return nil
}

func (rec *Recorder) RunCommand(ctx context.Context, command string) error {
	mark := rec.Transcript
	stmt, err := parseStmt(command)
	if err != nil {
		return fmt.Errorf("parsing: %w", err)
	}
	fmt.Fprintf(&rec.Transcript, "$ %s\n", command)
	runErr := rec.runner.Run(ctx, stmt)
	if err := rec.flush(); err != nil {
		return err
	}
	if status, ok := interp.IsExitStatus(runErr); ok {
		fmt.Fprintf(&rec.Transcript, "? %d\n", status)
	}
	if rec.runner.Exited() {
		rec.Transcript = mark
	}
	return runErr
}

func (rec *Recorder) Exited() bool {
	return rec.runner.Exited()
}

func parseStmt(s string) (syntax.Node, error) {
	r := strings.NewReader(s)
	f, err := syntax.NewParser().Parse(r, "")
	if err != nil {
		return nil, err
	}
	if err == nil && len(f.Stmts) != 1 {
		return nil, fmt.Errorf("expected exactly one statement, got %d", len(f.Stmts))
	}
	return f.Stmts[0], nil
}
