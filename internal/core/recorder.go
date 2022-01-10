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
		W: &prefixingWriter{
			Prefix: "1 ",
			W:      &rec.Transcript,
		},
	}
	rec.stderr = &lineBufferingWriter{
		W: &prefixingWriter{
			Prefix: "2 ",
			W:      &rec.Transcript,
		},
	}
	rec.runner, err = interp.New(
		interp.StdIO(nil,
			io.MultiWriter(rec.stdout, rec.Stdout),
			io.MultiWriter(rec.stderr, rec.Stderr),
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

type CommandResult struct {
	Output   []byte
	ExitCode int
}

func (rec *Recorder) RunCommand(ctx context.Context, command string) (*CommandResult, error) {
	// Record command.
	beforeCommandMark := rec.Transcript.Len()
	stmt, err := parseStmt(command)
	if err != nil {
		return nil, fmt.Errorf("parsing: %w", err)
	}
	if rec.Transcript.Len() > 0 {
		fmt.Fprintln(&rec.Transcript)
	}
	fmt.Fprintf(&rec.Transcript, "$ %s\n", command)
	afterCommandMark := rec.Transcript.Len()

	// Execute command and record output.
	runErr := rec.runner.Run(ctx, stmt)
	if err := rec.flush(); err != nil {
		return nil, err
	}
	var res CommandResult
	res.Output = rec.Transcript.Bytes()[afterCommandMark:rec.Transcript.Len()]

	// Record exit code.
	if status, ok := interp.IsExitStatus(runErr); ok {
		res.ExitCode = int(status)
		fmt.Fprintf(&rec.Transcript, "? %d\n", status)
		runErr = nil
	}

	// Assume final command is simply "exit", so exclude it from transcript.
	// TODO: Validate this assumption.
	if rec.runner.Exited() {
		rec.Transcript.Truncate(beforeCommandMark)
	}

	if runErr != nil {
		return nil, runErr
	}
	return &res, nil
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
