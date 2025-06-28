package core

import (
	"bytes"
	"io"
	"strings"

	"github.com/akedrou/textdiff"
	"github.com/fatih/color"
)

type CommandCheckError struct {
	Command string
	Lineno  int
	Errs    []error
}

func (err CommandCheckError) Error() string {
	return "command checks failed"
}

type DiffError struct {
	Expected string
	Actual   string
}

func (err DiffError) Error() string {
	return "output differs"
}

func (err DiffError) Plain() string {
	return textdiff.Unified("expected", "actual", err.Expected, err.Actual)
}

func (err DiffError) Color() string {
	var buf bytes.Buffer
	for line := range strings.Lines(err.Plain()) {
		switch {
		case strings.HasPrefix(line, "---"):
			yellow.Fprintf(&buf, "%s", line)
		case strings.HasPrefix(line, "+++"):
			yellow.Fprintf(&buf, "%s", line)
		case strings.HasPrefix(line, "@@ "):
			cyan.Fprintf(&buf, "%s", line)
		case strings.HasPrefix(line, "+"):
			green.Fprintf(&buf, "%s", line)
		case strings.HasPrefix(line, "-"):
			red.Fprintf(&buf, "%s", line)
		default:
			io.WriteString(&buf, line)
		}
	}
	return buf.String()
}

var yellow = color.New(color.FgYellow)
var cyan = color.New(color.FgCyan)
var green = color.New(color.FgGreen)
var red = color.New(color.FgRed)

