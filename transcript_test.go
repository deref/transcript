// This file exercises the Go API.
// It's intentionally redundant with ./test.sh, which exercises the CLI.

package main_test

import (
	"embed"
	"io/fs"
	"os"
	"strings"
	"testing"

	"github.com/deref/transcript/cmdtest"
	"github.com/stretchr/testify/assert"
)

//go:embed tests/*
var tests embed.FS

func TestTranscript(t *testing.T) {
	// Change to tests directory to match behavior of test.sh
	err := os.Chdir("tests")
	if !assert.NoError(t, err) {
		return
	}
	defer os.Chdir("..")

	err = fs.WalkDir(tests, "tests", func(path string, d fs.DirEntry, err error) error {
		if !assert.NoError(t, err) {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(path, ".cmdt") {
			return nil
		}
		f, err := tests.Open(path)
		if !assert.NoError(t, err) {
			return nil
		}
		defer f.Close()
		t.Run("check:"+path, func(t *testing.T) {
			cmdtest.Check(t, f)
		})
		return nil
	})
	assert.NoError(t, err)
}
