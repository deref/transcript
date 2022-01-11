package core

import (
	"bytes"
	"context"
	"fmt"
	"io"
)

type Updater struct {
	rec    *Recorder
	lineno int
}

func (upr *Updater) UpdateTranscript(ctx context.Context, r io.Reader) (transcript *bytes.Buffer, err error) {
	upr.rec = &Recorder{}
	if err := upr.rec.Init(); err != nil {
		return nil, fmt.Errorf("initializing recorder: %w", err)
	}
	interp := &Interpreter{
		Handler: &updateHandler{
			Updater: upr,
		},
	}
	if err := interp.ExecTranscript(ctx, r); err != nil {
		return nil, err
	}
	return &upr.rec.Transcript, nil
}

type updateHandler struct {
	*Updater
}

func (upr *updateHandler) HandleComment(ctx context.Context, text string) error {
	upr.rec.RecordComment(text)
	return nil
}

func (upr *updateHandler) HandleRun(ctx context.Context, command string) error {
	_, err := upr.rec.RunCommand(ctx, command)
	return err
}

func (upr *updateHandler) HandleOutput(ctx context.Context, fd int, line string) error {
	return nil
}

func (upr *updateHandler) HandleExitCode(ctx context.Context, exitCode int) error {
	return nil
}
