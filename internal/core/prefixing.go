package core

import (
	"bytes"
	"io"
)

// An io.Writer that inserts a prefix at the start of each new line written to W.
// The prefix is the combination of a Label and a Separator. The separator is only
// printed if the line is non-empty.
type prefixingWriter struct {
	label     string
	separator string
	w         io.Writer

	prev prevByte
}

// Negative values are special cases.
// Non-negative values are the previously written byte.
type prevByte int

const (
	prevNone prevByte = -1
)

func newPrefixingWriter(label, separator string, w io.Writer) *prefixingWriter {
	return &prefixingWriter{
		label:     label,
		separator: separator,
		w:         w,

		prev: prevNone,
	}
}

func (w *prefixingWriter) Write(bs []byte) (int, error) {
	l := 0
	for r, b := range bs {
		switch w.prev {
		case prevNone, '\n':
			n, err := w.w.Write(bs[l:r])
			if err != nil {
				return r + n, err
			}
			l = r
			if _, err := io.WriteString(w.w, w.label); err != nil {
				return r - 1, err
			}
			if b != '\n' {
				if _, err := io.WriteString(w.w, w.separator); err != nil {
					return r - 1, err
				}
			}
		}
		w.prev = prevByte(b)
	}
	n, err := w.w.Write(bs[l:])
	return l + n, err
}

// An io.Writer that only writes complete lines to the underlying W.
type lineBufferingWriter struct {
	W io.Writer

	buf bytes.Buffer
}

// Call when finished writing. If there is an incomplete line buffered,
// flush it _without_ appending a trailing new line.
func (w *lineBufferingWriter) Flush() error {
	_, err := w.flushBuffer()
	return err
}

func (w *lineBufferingWriter) flushBuffer() (n int, err error) {
	n, err = w.W.Write(w.buf.Bytes())
	w.buf.Next(n)
	return
}

func (w *lineBufferingWriter) Write(bs []byte) (int, error) {
	for i, b := range bs {
		w.buf.WriteByte(b)
		if b == '\n' {
			if _, err := w.flushBuffer(); err != nil {
				return i + 1, err
			}
		}
	}
	return len(bs), nil
}
