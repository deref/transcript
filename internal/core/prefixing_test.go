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
