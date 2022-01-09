package core

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPrefixingWriter(t *testing.T) {
	var buf bytes.Buffer
	w := &prefixingWriter{
		Prefix: ">>",
		W:      &buf,
	}
	n, err := w.Write([]byte("abc\nxyz"))
	assert.NoError(t, err)
	assert.Equal(t, n, 7)
	assert.Equal(t, ">>abc\n>>xyz", buf.String())
}

func TestLineBufferingWriter(t *testing.T) {
	var buf bytes.Buffer

	lbw1 := &lineBufferingWriter{W: &buf}
	pw1 := &prefixingWriter{
		Prefix: "1 ",
		W:      lbw1,
	}

	lbw2 := &lineBufferingWriter{W: &buf}
	pw2 := &prefixingWriter{
		Prefix: "2 ",
		W:      lbw2,
	}

	pw1.Write([]byte("a")) // Start a line.
	pw2.Write([]byte("x")) // Incomplete line.
	assert.Equal(t, "", buf.String())
	pw1.Write([]byte("b\n")) // Complete the line.
	assert.Equal(t, "1 ab\n", buf.String())
	pw1.Write([]byte("c\n")) // End with a complete line.
	lbw1.Flush()             // No-op.
	lbw2.Flush()             // Completes line.
	assert.Equal(t, "1 ab\n1 c\n2 x\n", buf.String())
}
