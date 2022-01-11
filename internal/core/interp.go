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
	Lineno        int
	Command       string
	CommandLineno int

	// Private state.
	acceptResults bool
}

type Handler interface {
	HandleComment(ctx context.Context, text string) error
	HandleRun(ctx context.Context, command string) error
	HandleOutput(ctx context.Context, fd int, line string) error
	HandleExitCode(ctx context.Context, exitCode int) error
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
	return nil
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
		t.Command = payload
		t.CommandLineno = t.Lineno
		t.acceptResults = true
		return hdlr.HandleRun(ctx, payload)

	case "1", "2":
		if !t.acceptResults {
			return t.syntaxErrorf("unexpected output check")
		}
		return hdlr.HandleOutput(ctx, int(opcode[0])-'1'+1, payload)

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

	default:
		return t.syntaxErrorf("invalid opcode: %q", text[0])
	}
}

func (t *Interpreter) syntaxErrorf(message string, v ...interface{}) error {
	return fmt.Errorf("on line %d: "+message, append([]interface{}{t.Lineno}, v...))
}
