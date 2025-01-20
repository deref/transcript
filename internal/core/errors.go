package core

import (
	"bytes"
	"fmt"
	"io"
	"iter"
	"strings"

	"github.com/akedrou/textdiff"
	"github.com/fatih/color"
)

type CommandCheckError struct {
	Command string
	Lineno  int
	Err     error
}

func (err CommandCheckError) Unwrap() error {
	return err.Err
}

func (err CommandCheckError) Error() string {
	return fmt.Sprintf("check failed: %v", err.Err)
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
	for line := range lines(err.Plain()) {
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

// Borrowed from the future: Go 1.24 gains this function.
//
// Lines returns an iterator over the newline-terminated lines in the string s.
// The lines yielded by the iterator include their terminating newlines.
// If s is empty, the iterator yields no lines at all.
// If s does not end in a newline, the final yielded line will not end in a newline.
// It returns a single-use iterator.
func lines(s string) iter.Seq[string] {
	return func(yield func(string) bool) {
		for len(s) > 0 {
			var line string
			if i := strings.IndexByte(s, '\n'); i >= 0 {
				line, s = s[:i+1], s[i+1:]
			} else {
				line, s = s, ""
			}
			if !yield(line) {
				return
			}
		}
		return
	}
}
