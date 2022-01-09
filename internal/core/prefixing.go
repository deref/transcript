package core

import (
	"bytes"
	"io"
)

// An io.Writer that inserts Prefix at the start of each new line written to W.
type prefixingWriter struct {
	Prefix string
	W      io.Writer

	prev byte
}

func (w *prefixingWriter) Write(bs []byte) (int, error) {
	l := 0
	for r, b := range bs {
		switch w.prev {
		case 0, '\n':
			n, err := w.W.Write(bs[l:r])
			if err != nil {
				return r + n, err
			}
			l = r
			if _, err := io.WriteString(w.W, w.Prefix); err != nil {
				return r - 1, err
			}
		}
		w.prev = b
	}
	n, err := w.W.Write(bs[l:])
	return l + n, err
}

// An io.Writer that only writes complete lines to the underlying W.
type lineBufferingWriter struct {
	W io.Writer

	buf bytes.Buffer
}

// Call when finished writing. If there is an incomplete line buffered,
// flush it and append a trailing new line.
func (w *lineBufferingWriter) Flush() error {
	n, err := w.flushBuffer()
	if err == nil && n > 0 {
		_, err = w.W.Write([]byte{'\n'})
	}
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
