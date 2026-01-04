package core

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"mvdan.cc/sh/v3/expand"
	"mvdan.cc/sh/v3/interp"
	"mvdan.cc/sh/v3/syntax"
)

// lookupEnv provides an environment backed by os.LookupEnv/os.Environ.
//
// This is important for `go test` caching: environment variables consulted by
// the test process (via os.LookupEnv) are recorded in the "testlog" and become
// part of the cache key. If we used a snapshot like expand.ListEnviron(os.Environ()...),
// variable expansions would not necessarily be visible to the test cache.
type lookupEnv struct{}

func (lookupEnv) Get(name string) expand.Variable {
	v, ok := os.LookupEnv(name)
	if !ok {
		return expand.Variable{}
	}
	return expand.Variable{
		Kind:     expand.String,
		Str:      v,
		Exported: true,
	}
}

func (lookupEnv) Each(fn func(name string, vr expand.Variable) bool) {
	for _, kv := range os.Environ() {
		name, value, ok := strings.Cut(kv, "=")
		if !ok {
			continue
		}
		if !fn(name, expand.Variable{
			Kind:     expand.String,
			Str:      value,
			Exported: true,
		}) {
			return
		}
	}
}

// Recorder is a shell Interpreter that captures a command transcript
// into the Transcript byte buffer.
type Recorder struct {
	// If provided, tees Stdout to this writer in addition to the buffer.
	Stdout io.Writer
	// If provided, tees Stderr to this writer in addition to the buffer.
	Stderr io.Writer
	// Transcript captures the recorded output in cmdt format.
	Transcript bytes.Buffer

	needsBlank     bool
	runner         *interp.Runner
	stdoutBuf      bytes.Buffer
	stderrBuf      bytes.Buffer
	fileCount      int      // Counter for auto-generated binary file names
	preferredFiles []string // List of preferred filenames in order (stderr first, then stdout)
	fileIndex      int      // Current position in preferredFiles slice
}

func (rec *Recorder) Init() error {
	var err error
	rec.runner, err = interp.New(
		interp.Env(lookupEnv{}),
		interp.ExecHandlers(func(next interp.ExecHandlerFunc) interp.ExecHandlerFunc {
			return func(ctx context.Context, args []string) error {
				if len(args) > 0 && args[0] == "dep" {
					return runDepIntrinsic(ctx, args[1:])
				}
				return next(ctx, args)
			}
		}),
		interp.StdIO(nil,
			io.MultiWriter(&rec.stdoutBuf, orDiscard(rec.Stdout)),
			io.MultiWriter(&rec.stderrBuf, orDiscard(rec.Stderr)),
		))
	rec.preferredFiles = make([]string, 0)
	rec.fileIndex = 0
	return err
}

func orDiscard(w io.Writer) io.Writer {
	if w == nil {
		return io.Discard
	}
	return w
}

// SetPreferredFiles sets the list of preferred filenames in order.
// Files should be provided in deterministic order (stderr first, then stdout).
func (rec *Recorder) SetPreferredFiles(files []string) {
	rec.preferredFiles = make([]string, len(files))
	copy(rec.preferredFiles, files)
	rec.fileIndex = 0
}

// generateBinaryFilename creates a filename, preferring existing names when available.
// Uses deterministic ordering (stderr first, then stdout) to consume preferred filenames.
func (rec *Recorder) generateBinaryFilename() string {
	// Check if we have a preferred filename available.
	if rec.fileIndex < len(rec.preferredFiles) {
		filename := rec.preferredFiles[rec.fileIndex]
		rec.fileIndex++
		return filename
	}

	// Fall back to auto-generated filename.
	rec.fileCount++
	return fmt.Sprintf("%03d.bin", rec.fileCount)
}

func (rec *Recorder) flush() error {
	// Write stderr first (usually empty, text-only, important not to miss).
	if err := rec.flushBuffer(&rec.stderrBuf, 2); err != nil {
		return err
	}
	// Then write stdout.
	if err := rec.flushBuffer(&rec.stdoutBuf, 1); err != nil {
		return err
	}
	return nil
}

// flushBuffer processes output from a command and writes it to the transcript.
// Individual command outputs are expected to be reasonably small (not streaming large files).
func (rec *Recorder) flushBuffer(buf *bytes.Buffer, fd int) error {
	if buf.Len() == 0 {
		return nil
	}

	data := buf.Bytes()
	buf.Reset()

	// Check if data is binary.
	if isBinary(data) {
		// Write binary data to file and reference it.
		filename := rec.generateBinaryFilename()
		if err := os.WriteFile(filename, data, 0644); err != nil {
			return fmt.Errorf("writing binary file %q: %w", filename, err)
		}
		fmt.Fprintf(&rec.Transcript, "%d< %s\n", fd, filename)
		return nil
	}

	// Handle text output - add prefix to each line and write to transcript.
	for line := range bytes.Lines(data) {
		if len(line) == 1 && line[0] == '\n' {
			// Empty line - just prefix.
			fmt.Fprintf(&rec.Transcript, "%d\n", fd)
		} else {
			// Non-empty line - prefix + space + line.
			fmt.Fprintf(&rec.Transcript, "%d %s", fd, line)
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

func (rec *Recorder) RunDepDirective(ctx context.Context, payload string) error {
	stmt, err := parseStmt("dep " + payload)
	if err != nil {
		return fmt.Errorf("parsing: %w", err)
	}
	if err := validateDepStmt(stmt); err != nil {
		return err
	}
	runErr := rec.runner.Run(ctx, stmt)
	// The intrinsic should be silent. Discard any output to avoid polluting the
	// next recorded command.
	rec.stdoutBuf.Reset()
	rec.stderrBuf.Reset()
	if runErr != nil {
		if status, ok := interp.IsExitStatus(runErr); ok {
			return fmt.Errorf("dep exited with status %d", status)
		}
		return runErr
	}
	return nil
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
