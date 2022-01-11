package core

import (
	"bytes"
	"context"
	"fmt"
	"io"
)

type Checker struct {
	rec              *Recorder
	interpreter      *Interpreter
	expectedOutput   bytes.Buffer
	expectedExitCode int
	actualResult     *CommandResult
}

func (ckr *Checker) CheckTranscript(ctx context.Context, r io.Reader) error {
	ckr.rec = &Recorder{}
	if err := ckr.rec.Init(); err != nil {
		return fmt.Errorf("initializing recorder: %w", err)
	}

	ckr.interpreter = &Interpreter{
		Handler: &checkHandler{
			Checker: ckr,
		},
	}

	if err := ckr.interpreter.ExecTranscript(ctx, r); err != nil {
		return err
	}
	return ckr.endCommand()
}

type checkHandler struct {
	*Checker
}

func (ckr *checkHandler) HandleComment(ctx context.Context, text string) error {
	return nil
}

func (ckr *checkHandler) HandleRun(ctx context.Context, command string) error {
	if err := ckr.endCommand(); err != nil {
		return err
	}
	var err error
	ckr.actualResult, err = ckr.rec.RunCommand(ctx, command)
	if err != nil {
		return ckr.commandCheckError(err)
	}
	return nil
}

func (ckr *checkHandler) HandleOutput(ctx context.Context, fd int, line string) error {
	return ckr.expectOutput(fmt.Sprintf("%d %s", fd, line))
}

func (ckr *checkHandler) HandleExitCode(ctx context.Context, exitCode int) error {
	ckr.expectedExitCode = exitCode
	return nil
}

func (ckr *Checker) endCommand() error {
	if ckr.actualResult == nil {
		return nil
	}
	if ckr.expectedExitCode != ckr.actualResult.ExitCode {
		return ckr.commandCheckError(fmt.Errorf("expected exit code %d, but got %d", ckr.expectedExitCode, ckr.actualResult.ExitCode))
	}

	expectedOutput := ckr.expectedOutput.String()
	actualOutput := string(ckr.actualResult.Output)
	if expectedOutput != actualOutput {
		return ckr.commandCheckError(DiffError{
			Expected: expectedOutput,
			Actual:   actualOutput,
		})
	}

	ckr.expectedOutput.Reset()
	ckr.expectedExitCode = 0

	return nil
}

func (ckr *Checker) expectOutput(text string) error {
	fmt.Fprintln(&ckr.expectedOutput, text)
	return nil
}

func (ckr *Checker) syntaxErrorf(message string, v ...interface{}) error {
	return fmt.Errorf("on line %d: "+message, append([]interface{}{ckr.interpreter.Lineno}, v...))
}

func (ckr *Checker) commandCheckError(err error) CommandCheckError {
	return CommandCheckError{
		Command: ckr.interpreter.Command,
		Lineno:  ckr.interpreter.CommandLineno,
		Err:     err,
	}
}
