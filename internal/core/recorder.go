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
	stdoutBuf  bytes.Buffer
	stderrBuf  bytes.Buffer
}

func (rec *Recorder) Init() error {
	var err error
	rec.runner, err = interp.New(
		interp.StdIO(nil,
			io.MultiWriter(&rec.stdoutBuf, orDiscard(rec.Stdout)),
			io.MultiWriter(&rec.stderrBuf, orDiscard(rec.Stderr)),
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
	// Write stderr first (usually empty, text-only, important not to miss).
	if err := rec.flushBuffer(&rec.stderrBuf, "2"); err != nil {
		return err
	}
	// Then write stdout.
	if err := rec.flushBuffer(&rec.stdoutBuf, "1"); err != nil {
		return err
	}
	return nil
}

func (rec *Recorder) flushBuffer(buf *bytes.Buffer, prefix string) error {
	if buf.Len() == 0 {
		return nil
	}
	
	data := buf.Bytes()
	buf.Reset()
	
	// Add prefix to each line and write to transcript.
	for line := range bytes.Lines(data) {
		if len(line) == 1 && line[0] == '\n' {
			// Empty line - just prefix.
			fmt.Fprintf(&rec.Transcript, "%s\n", prefix)
		} else {
			// Non-empty line - prefix + space + line.
			fmt.Fprintf(&rec.Transcript, "%s %s", prefix, line)
		}
	}
	
	// Handle case where original didn't end with newline.
	if len(data) > 0 && data[len(data)-1] != '\n' {
		io.WriteString(&rec.Transcript, "\n% no-newline\n")
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
