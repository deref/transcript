package core

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"strconv"
	"strings"
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
}

type Handler interface {
	HandleComment(ctx context.Context, text string) error
	HandleRun(ctx context.Context, command string) error
	HandleOutput(ctx context.Context, fd int, line string) error
	HandleNoNewline(ctx context.Context, fd int) error
	HandleExitCode(ctx context.Context, exitCode int) error
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

	return t.flushCommand(ctx)
}

func (t *Interpreter) ExecLine(ctx context.Context, text string) error {
	hdlr := t.Handler
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
		t.Command = payload
		t.CommandLineno = t.Lineno
		t.acceptResults = true
		return hdlr.HandleRun(ctx, payload)

	case "1", "2":
		if !t.acceptResults {
			return t.syntaxErrorf("unexpected output check")
		}
		fd := int(opcode[0]) - '1' + 1
		t.prevFD = fd
		return hdlr.HandleOutput(ctx, fd, payload)

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

		default:
			return t.syntaxErrorf("invalid directive: %q", directive)
		}

	default:
		return t.syntaxErrorf("invalid opcode: %q", text[0])
	}
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
