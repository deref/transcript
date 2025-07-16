package core

import (
	"bytes"
	"context"
	"io"
	"strconv"
	"strings"
)

type Formatter struct {
	buf *bytes.Buffer
}

func (f *Formatter) FormatTranscript(ctx context.Context, r io.Reader) (transcript *bytes.Buffer, err error) {
	f.buf = &bytes.Buffer{}

	// Use the regular interpreter with the formatter as handler.
	interp := &Interpreter{
		Handler: f,
	}
	if err := interp.ExecTranscript(ctx, r); err != nil {
		return nil, err
	}

	// Ensure file ends with exactly one newline
	content := f.buf.Bytes()
	content = bytes.TrimRight(content, "\n")
	if len(content) > 0 {
		content = append(content, '\n')
	}

	return bytes.NewBuffer(content), nil
}

func (f *Formatter) HandleComment(ctx context.Context, text string) error {
	// Normalize comments and blank lines
	trimmed := strings.TrimSpace(text)
	if trimmed == "" {
		// Blank line
		f.buf.WriteString("\n")
	} else if strings.HasPrefix(trimmed, "#") {
		// Normalize comment formatting
		comment := strings.TrimPrefix(trimmed, "#")
		comment = strings.TrimSpace(comment)
		if comment == "" {
			f.buf.WriteString("#\n")
		} else {
			f.buf.WriteString("# " + comment + "\n")
		}
	}

	return nil
}

func (f *Formatter) HandleRun(ctx context.Context, command string) error {
	// Write the command with normalized formatting
	f.buf.WriteString("$ " + strings.TrimSpace(command) + "\n")

	return nil
}

func (f *Formatter) HandleOutput(ctx context.Context, fd int, line string) error {
	// Output lines preserve their exact content (including whitespace)
	f.buf.WriteString(strconv.Itoa(fd) + " " + line + "\n")

	return nil
}

func (f *Formatter) HandleFileOutput(ctx context.Context, fd int, filepath string) error {
	// File output references with normalized formatting
	f.buf.WriteString(strconv.Itoa(fd) + "< " + strings.TrimSpace(filepath) + "\n")

	return nil
}

func (f *Formatter) HandleNoNewline(ctx context.Context, fd int) error {
	// No-newline directive with normalized formatting
	f.buf.WriteString("% no-newline\n")

	return nil
}

func (f *Formatter) HandleExitCode(ctx context.Context, exitCode int) error {
	// Exit code with normalized formatting
	f.buf.WriteString("? " + strconv.Itoa(exitCode) + "\n")

	return nil
}

func (f *Formatter) HandleEnd(ctx context.Context) error {
	// No special handling needed for end
	return nil
}
