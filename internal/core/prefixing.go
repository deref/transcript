package core

import (
	"io"
)

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
