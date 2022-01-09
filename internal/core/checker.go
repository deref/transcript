package core

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/sergi/go-diff/diffmatchpatch"
)

type Checker struct {
	lineno int

	rec *Recorder

	command          string
	commandLineno    int
	expectedOutput   bytes.Buffer
	expectedExitCode int
	actualResult     *CommandResult
}

func (ckr *Checker) CheckTranscript(ctx context.Context, r io.Reader) error {
	ckr.rec = &Recorder{}
	if err := ckr.rec.Init(); err != nil {
		return fmt.Errorf("initializing recorder: %w", err)
	}

	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		ckr.lineno++
		if err := ckr.checkLine(ctx, scanner.Text()); err != nil {
			return err
		}
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("scanning: %w", err)
	}
	return ckr.endCommand()
}

func (ckr *Checker) checkLine(ctx context.Context, text string) error {
	if text == "" || text[0] == '#' {
		return nil
	}
	parts := strings.SplitN(text, " ", 2)
	opcode := parts[0]
	var payload string
	if len(parts) == 2 {
		payload = parts[1]
	}
	switch opcode {

	case "$":
		if err := ckr.endCommand(); err != nil {
			return err
		}
		ckr.command = payload
		ckr.commandLineno = ckr.lineno
		var err error
		ckr.actualResult, err = ckr.rec.RunCommand(ctx, ckr.command)
		if err != nil {
			return ckr.commandCheckError(err)
		}
		return nil

	case "1", "2":
		return ckr.expectOutput(text)

	case "?":
		if ckr.command == "" {
			return ckr.syntaxErrorf("unexpected exit status check")
		}
		var err error
		ckr.expectedExitCode, err = strconv.Atoi(payload)
		if err != nil {
			return ckr.syntaxErrorf("parsing error code: %w", err)
		}
		return nil

	default:
		return ckr.syntaxErrorf("invalid opcode: %q", ckr.lineno, text[0])
	}
}

func (ckr *Checker) endCommand() error {
	if ckr.command == "" {
		return nil
	}
	if ckr.expectedExitCode != ckr.actualResult.ExitCode {
		return ckr.commandCheckError(fmt.Errorf("expected exit code %d, but got %d", ckr.expectedExitCode, ckr.actualResult.ExitCode))
	}

	dmp := diffmatchpatch.New()
	diffs := dmp.DiffMain(ckr.expectedOutput.String(), string(ckr.actualResult.Output), true)
	if differs(diffs) {
		return ckr.commandCheckError(DiffError{
			Diffs: diffs,
		})
	}

	ckr.command = ""
	ckr.expectedOutput.Reset()
	ckr.expectedExitCode = 0

	return nil
}

func differs(diffs []diffmatchpatch.Diff) bool {
	return len(diffs) > 1 || diffs[0].Type != diffmatchpatch.DiffEqual
}

func (ckr *Checker) expectOutput(text string) error {
	if ckr.command == "" {
		return ckr.syntaxErrorf("unexpected output check")
	}
	fmt.Fprintln(&ckr.expectedOutput, text)
	return nil
}

func (ckr *Checker) syntaxErrorf(message string, v ...interface{}) error {
	return fmt.Errorf("on line %d: "+message, append([]interface{}{ckr.lineno}, v...))
}

func (ckr *Checker) commandCheckError(err error) CommandCheckError {
	return CommandCheckError{
		Command: ckr.command,
		Lineno:  ckr.lineno,
		Err:     err,
	}
}
