package cmdtest

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/deref/transcript/internal/core"
	"github.com/stretchr/testify/assert"
)

func Check(t *testing.T, r io.Reader) (ok bool) {
	ckr := &core.Checker{}
	err := ckr.CheckTranscript(context.TODO(), r)
	var chkErr core.CommandCheckError
	if errors.As(err, &chkErr) {
		t.Logf("failed check on line %d:", chkErr.Lineno)
		t.Logf("$ %s", chkErr.Command)
		t.Log(chkErr.Err.Error())
		var diffErr core.DiffError
		if errors.As(err, &diffErr) {
			t.Log(diffErr.Plain())
		}
		t.Fail()
		return false
	}
	return assert.NoError(t, err)
}

func CheckString(t *testing.T, cmdt string) bool {
	return Check(t, strings.NewReader(cmdt))
}
