package main_test

import (
	"embed"
	"io/fs"
	"strings"
	"testing"

	"github.com/deref/transcript/cmdtest"
	"github.com/stretchr/testify/assert"
)

//go:embed tests/*
var tests embed.FS

func TestTranscript(t *testing.T) {
	err := fs.WalkDir(tests, ".", func(path string, d fs.DirEntry, err error) error {
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
		t.Run("checking_"+path, func(t *testing.T) {
			cmdtest.Check(t, f)
		})
		return nil
	})
	assert.NoError(t, err)
}
