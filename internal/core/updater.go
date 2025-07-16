package core

import (
	"bytes"
	"context"
	"fmt"
	"io"
)

type Updater struct {
	rec            *Recorder
	lineno         int
	fileRefs       []string // File references for current command
	currentCommand string
}

func (upr *Updater) UpdateTranscript(ctx context.Context, r io.Reader) (transcript *bytes.Buffer, err error) {
	// Initialize recorder for streaming processing.
	upr.rec = &Recorder{}
	if err := upr.rec.Init(); err != nil {
		return nil, fmt.Errorf("initializing recorder: %w", err)
	}

	// Use the regular interpreter with the updater as handler.
	interp := &Interpreter{
		Handler: upr,
	}
	if err := interp.ExecTranscript(ctx, r); err != nil {
		return nil, err
	}
	return &upr.rec.Transcript, nil
}

func (upr *Updater) flushCurrentCommand(ctx context.Context) error {
	if upr.currentCommand == "" {
		return nil // No command to flush
	}

	// Set up recorder with file references for this command
	upr.rec.SetPreferredFiles(upr.fileRefs)

	// Execute the command
	if _, err := upr.rec.RunCommand(ctx, upr.currentCommand); err != nil {
		return err
	}

	// Clear the buffer
	upr.fileRefs = nil
	upr.currentCommand = ""

	return nil
}

func (upr *Updater) HandleComment(ctx context.Context, text string) error {
	// Flush any buffered command before processing comments to maintain order
	if err := upr.flushCurrentCommand(ctx); err != nil {
		return err
	}
	// Comments are processed immediately, not buffered
	upr.rec.RecordComment(text)
	return nil
}

func (upr *Updater) HandleRun(ctx context.Context, command string) error {
	// Flush any previous command before starting a new one
	if err := upr.flushCurrentCommand(ctx); err != nil {
		return err
	}

	// Buffer this command - don't execute it yet
	upr.currentCommand = command
	return nil
}

func (upr *Updater) HandleOutput(ctx context.Context, fd int, line string) error {
	// Output lines are ignored in update mode - we only care about file references
	return nil
}

func (upr *Updater) HandleFileOutput(ctx context.Context, fd int, filepath string) error {
	// Collect file references for the current command
	upr.fileRefs = append(upr.fileRefs, filepath)
	return nil
}

func (upr *Updater) HandleNoNewline(ctx context.Context, fd int) error {
	// No-newline directives are ignored in update mode
	return nil
}

func (upr *Updater) HandleExitCode(ctx context.Context, exitCode int) error {
	// Exit codes are ignored in update mode, but flush the command now that we have all its output
	return upr.flushCurrentCommand(ctx)
}

func (upr *Updater) HandleEnd(ctx context.Context) error {
	// Flush any remaining command at the end
	return upr.flushCurrentCommand(ctx)
}
