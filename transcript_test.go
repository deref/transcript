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
	// Save original directory
	originalDir, err := os.Getwd()
	if !assert.NoError(t, err) {
		return
	}
	defer os.Chdir(originalDir)

	err = fs.WalkDir(tests, "tests", func(path string, d fs.DirEntry, err error) error {
		if !assert.NoError(t, err) {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(path, "test.cmdt") {
			return nil
		}

		// Extract test directory from path (e.g., "tests/binary-output/test.cmdt" -> "tests/binary-output")
		testDir := strings.TrimSuffix(path, "/test.cmdt")

		f, err := tests.Open(path)
		if !assert.NoError(t, err) {
			return nil
		}
		defer f.Close()
		t.Run("check:"+path, func(t *testing.T) {
			// Change to the test directory before running the test
			err := os.Chdir(testDir)
			if !assert.NoError(t, err) {
				return
			}
			defer os.Chdir(originalDir)

			cmdtest.Check(t, f)
		})
		return nil
	})
	assert.NoError(t, err)
}
