package core

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"

	"mvdan.cc/sh/v3/syntax"
)

// Interprets a transcript file.
type Interpreter struct {
	// Input parameters.
	Handler Handler

	// Exposed state.
	Lineno        int    // Line currently executing.
	Command       string // Text of the most recently executed command.
	CommandLineno int    // Line of the most recently executed command.

	// Private state.
	acceptResults bool
	prevFD        int // stdout (1), stderr (2) or none (0).

	pendingCommandLineno int
	pendingCommandLines  []string
}

// Handler provides callbacks for processing transcript operations.
// Each method corresponds to a specific cmdt opcode or interpreter event.
type Handler interface {
	// HandleComment processes comment lines and blank lines.
	// Corresponds to cmdt syntax: "# comment text" or blank lines.
	HandleComment(ctx context.Context, text string) error

	// HandleRun executes a shell command.
	// Corresponds to cmdt syntax: "$ command args".
	HandleRun(ctx context.Context, command string) error

	// HandleOutput processes expected output from a command.
	// The fd parameter indicates the file descriptor: 1 for stdout, 2 for stderr.
	// Corresponds to cmdt syntax: "1 stdout line" or "2 stderr line".
	HandleOutput(ctx context.Context, fd int, line string) error

	// HandleFileOutput processes expected output that references an external file.
	// The fd parameter indicates the file descriptor: 1 for stdout, 2 for stderr.
	// The filepath parameter specifies the file containing the expected output.
	// Corresponds to cmdt syntax: "1< filename" or "2< filename".
	HandleFileOutput(ctx context.Context, fd int, filepath string) error

	// HandleNoNewline indicates that the last output line did not end with a newline.
	// The fd parameter indicates which stream (stdout=1, stderr=2) lacks the newline.
	// Corresponds to cmdt syntax: "% no-newline".
	HandleNoNewline(ctx context.Context, fd int) error

	// HandleDep declares external dependencies for caching/correctness purposes.
	// The payload is interpreted as shell arguments/redirections to an intrinsic
	// `dep` command (for example: `foo "$BAR" < deps.txt`).
	//
	// Corresponds to cmdt syntax: "% dep <shell-args...>".
	HandleDep(ctx context.Context, payload string) error

	// HandleExitCode processes the expected exit code of a command.
	// If omitted in the transcript, the exit code defaults to 0.
	// Corresponds to cmdt syntax: "? exitcode".
	HandleExitCode(ctx context.Context, exitCode int) error

	// HandleEnd is called after a command and all its assertions have been processed.
	// This method has no direct cmdt syntax equivalent but signals command completion.
	HandleEnd(ctx context.Context) error
}

func (t *Interpreter) ExecTranscript(ctx context.Context, r io.Reader) error {
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		t.Lineno++
		if err := t.ExecLine(ctx, scanner.Text()); err != nil {
			return err
		}
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("scanning: %w", err)
	}
	if t.pendingCommandLineno != 0 {
		return t.syntaxErrorAtLinef(t.pendingCommandLineno, "unterminated command")
	}

	return t.flushCommand(ctx)
}

func (t *Interpreter) ExecLine(ctx context.Context, text string) error {
	hdlr := t.Handler
	if t.pendingCommandLineno != 0 {
		return t.appendCommandLine(ctx, text)
	}
	if strings.TrimSpace(text) == "" || text[0] == '#' {
		return hdlr.HandleComment(ctx, text)
	}
	parts := strings.SplitN(text, " ", 2)
	opcode := parts[0]
	var payload string
	if len(parts) == 2 {
		payload = parts[1]
	}
	switch opcode {
	case "$":
		if err := t.flushCommand(ctx); err != nil {
			return err
		}
		return t.startCommand(ctx, payload)

	case "1", "2":
		if !t.acceptResults {
			return t.syntaxErrorf("unexpected output check")
		}
		fd := int(opcode[0]) - '1' + 1
		t.prevFD = fd
		return hdlr.HandleOutput(ctx, fd, payload)

	case "1<", "2<":
		if !t.acceptResults {
			return t.syntaxErrorf("unexpected file output check")
		}
		fd := int(opcode[0]) - '1' + 1
		t.prevFD = fd
		return hdlr.HandleFileOutput(ctx, fd, payload)

	case "?":
		if !t.acceptResults {
			return t.syntaxErrorf("unexpected exit status check")
		}
		exitCode, err := strconv.Atoi(payload)
		if err != nil {
			return t.syntaxErrorf("parsing error code: %w", err)
		}
		err = hdlr.HandleExitCode(ctx, exitCode)
		t.acceptResults = false
		return err

	case "%":
		parts := strings.SplitN(payload, " ", 2)
		directive := parts[0]
		var payload string
		if len(parts) == 2 {
			payload = parts[1]
		}
		switch directive {

		case "no-newline":
			if t.prevFD == 0 {
				return t.syntaxErrorf("no output prior to no-newline")
			}
			if strings.TrimSpace(payload) != "" {
				return t.syntaxErrorf("unexpected arguments")
			}
			return hdlr.HandleNoNewline(ctx, t.prevFD)

		case "dep":
			if strings.TrimSpace(payload) == "" {
				return t.syntaxErrorf("usage: %% dep <shell-args...>")
			}
			return hdlr.HandleDep(ctx, payload)

		default:
			return t.syntaxErrorf("invalid directive: %q", directive)
		}

	default:
		return t.syntaxErrorf("invalid opcode: %q", text[0])
	}
}

func (t *Interpreter) startCommand(ctx context.Context, command string) error {
	t.pendingCommandLineno = t.Lineno
	t.pendingCommandLines = []string{command}
	return t.finishCommandIfComplete(ctx)
}

func (t *Interpreter) appendCommandLine(ctx context.Context, text string) error {
	t.pendingCommandLines = append(t.pendingCommandLines, text)
	return t.finishCommandIfComplete(ctx)
}

func (t *Interpreter) finishCommandIfComplete(ctx context.Context) error {
	command := strings.Join(t.pendingCommandLines, "\n")
	if commandIncomplete(command) {
		return nil
	}

	t.Command = command
	t.CommandLineno = t.pendingCommandLineno
	t.acceptResults = true
	t.pendingCommandLineno = 0
	t.pendingCommandLines = nil
	return t.Handler.HandleRun(ctx, command)
}

func commandIncomplete(command string) bool {
	_, err := parseStmt(command)
	if syntax.IsIncomplete(err) {
		return true
	}
	var parseErr syntax.ParseError
	return errors.As(err, &parseErr) && strings.HasPrefix(parseErr.Text, "unclosed here-document ")
}

func (t *Interpreter) flushCommand(ctx context.Context) error {
	if t.CommandLineno == 0 {
		return nil
	}
	t.prevFD = 0
	return t.Handler.HandleEnd(ctx)
}

func (t *Interpreter) syntaxErrorf(message string, v ...any) error {
	return fmt.Errorf("syntax error on line %d: "+message, append([]any{t.Lineno}, v...)...)
}

func (t *Interpreter) syntaxErrorAtLinef(lineno int, message string, v ...any) error {
	return fmt.Errorf("syntax error on line %d: "+message, append([]any{lineno}, v...)...)
}
