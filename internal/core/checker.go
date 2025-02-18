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

	return ckr.interpreter.ExecTranscript(ctx, r)
}

type checkHandler struct {
	*Checker
}

func (ckr *checkHandler) HandleComment(ctx context.Context, text string) error {
	return nil
}

func (ckr *checkHandler) HandleRun(ctx context.Context, command string) error {
	var err error
	ckr.actualResult, err = ckr.rec.RunCommand(ctx, command)
	if err != nil {
		return ckr.commandCheckError(err)
	}
	return nil
}

func (ckr *checkHandler) HandleOutput(ctx context.Context, fd int, line string) error {
	sep := ""
	if len(line) > 0 {
		sep = " "
	}
	return ckr.expectOutput(fmt.Sprintf("%d%s%s", fd, sep, line))
}

func (ckr *checkHandler) HandleNoNewline(ctx context.Context, fd int) error {
	// Assumes the previous line contains an already written newline.
	// This is also why we can ignore the fd parameter, as it's assumed to
	// match the previous line.
	_, err := io.WriteString(&ckr.expectedOutput, "% no-newline\n")
	return err
}

func (ckr *checkHandler) HandleExitCode(ctx context.Context, exitCode int) error {
	ckr.expectedExitCode = exitCode
	return nil
}

func (ckr *Checker) HandleEnd(ctx context.Context) error {
	defer func() {
		ckr.actualResult = nil
		ckr.expectedOutput.Reset()
		ckr.expectedExitCode = 0
	}()

	var errs []error

	expectedOutput := ckr.expectedOutput.String()
	actualOutput := string(ckr.actualResult.Output)
	if expectedOutput != actualOutput {
		//fmt.Printf("expected: %q\nactual: %q\n", expectedOutput, actualOutput)
		errs = append(errs, DiffError{
			Expected: expectedOutput,
			Actual:   actualOutput,
		})
	}

	if ckr.expectedExitCode != ckr.actualResult.ExitCode {
		errs = append(errs,
			fmt.Errorf("expected exit code %d, but got %d",
				ckr.expectedExitCode,
				ckr.actualResult.ExitCode))
	}

	if len(errs) > 0 {
		return ckr.commandCheckError(errs...)
	}
	return nil
}

func (ckr *Checker) expectOutput(text string) error {
	fmt.Fprintln(&ckr.expectedOutput, text)
	return nil
}

func (ckr *Checker) syntaxErrorf(message string, v ...any) error {
	return fmt.Errorf("syntax error on line %d: "+message, append([]any{ckr.interpreter.Lineno}, v...))
}

func (ckr *Checker) commandCheckError(errs ...error) CommandCheckError {
	return CommandCheckError{
		Command: ckr.interpreter.Command,
		Lineno:  ckr.interpreter.CommandLineno,
		Errs:    errs,
	}
}
