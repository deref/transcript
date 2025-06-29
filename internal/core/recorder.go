package core

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"strings"

	"mvdan.cc/sh/v3/interp"
	"mvdan.cc/sh/v3/syntax"
)

// Recorder is a shell Interpreter that captures a command transcript
// into the Transcript byte buffer.
type Recorder struct {
	// If provided, tees Stdout to this writer in addition to the buffer.
	Stdout io.Writer
	// If provided, tees Stderr to this writer in addition to the buffer.
	Stderr io.Writer
	// Transcript captures the recorded output in cmdt format.
	Transcript bytes.Buffer

	needsBlank bool
	runner     *interp.Runner
	stdout     *lineBufferingWriter
	stderr     *lineBufferingWriter
}

func (rec *Recorder) Init() error {
	var err error
	syncedTranscript := newSyncWriter(&rec.Transcript)
	rec.stdout = &lineBufferingWriter{
		W: newPrefixingWriter("1", " ", syncedTranscript),
	}
	rec.stderr = &lineBufferingWriter{
		W: newPrefixingWriter("2", " ", syncedTranscript),
	}
	rec.runner, err = interp.New(
		interp.StdIO(nil,
			io.MultiWriter(rec.stdout, orDiscard(rec.Stdout)),
			io.MultiWriter(rec.stderr, orDiscard(rec.Stderr)),
		))
	return err
}

func orDiscard(w io.Writer) io.Writer {
	if w == nil {
		return io.Discard
	}
	return w
}

func (rec *Recorder) flush() error {
	if err := rec.flushStream(rec.stdout); err != nil {
		return err
	}
	if err := rec.flushStream(rec.stderr); err != nil {
		return err
	}
	return nil
}

func (rec *Recorder) flushStream(w *lineBufferingWriter) error {
	if err := w.Flush(); err != nil {
		return err
	}
	n := rec.Transcript.Len()
	if n > 0 && rec.Transcript.Bytes()[n-1] != '\n' {
		w.Write([]byte{'\n'})
		io.WriteString(&rec.Transcript, "% no-newline\n")
	}
	return nil
}

type CommandResult struct {
	Output   []byte
	ExitCode int
}

func (rec *Recorder) RunCommand(ctx context.Context, command string) (*CommandResult, error) {
	stmt, err := parseStmt(command)
	if err != nil {
		return nil, fmt.Errorf("parsing: %w", err)
	}

	// Record command. Include a preceeding blank line for all but the first command.
	beforeCommandMark := rec.Transcript.Len()
	if rec.needsBlank {
		fmt.Fprintln(&rec.Transcript)
		rec.needsBlank = false
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
		rec.needsBlank = true
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

func (rec *Recorder) RecordComment(text string) {
	fmt.Fprintln(&rec.Transcript, text)
	rec.needsBlank = false
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
	if len(f.Stmts) != 1 {
		return nil, fmt.Errorf("expected exactly one statement, got %d", len(f.Stmts))
	}
	return f.Stmts[0], nil
}
