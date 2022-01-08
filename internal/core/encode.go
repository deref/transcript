package core

import (
	"fmt"
	"io"
)

type TranscriptEncoder struct {
	w io.Writer
}

func NewTranscriptEncoder(w io.Writer) *TranscriptEncoder {
	return &TranscriptEncoder{
		w: w,
	}
}

func (enc *TranscriptEncoder) EncodeComment(message string) error {
	_, err := fmt.Fprintf(enc.w, "# %s\n", message)
	return err
}

func (enc *TranscriptEncoder) EncodeCommand(command string) error {
	_, err := fmt.Fprintf(enc.w, "$ %s\n", command)
	return err
}

func (enc *TranscriptEncoder) EncodeOut(line string) error {
	_, err := fmt.Fprintf(enc.w, "1 %s\n", line)
	return err
}

func (enc *TranscriptEncoder) EncodeErr(line string) error {
	_, err := fmt.Fprintf(enc.w, "2 %s\n", line)
	return err
}

func (enc *TranscriptEncoder) EncodeExit(code int) error {
	_, err := fmt.Fprintf(enc.w, "? %d\n", code)
	return err
}
