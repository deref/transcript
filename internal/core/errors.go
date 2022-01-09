package core

import (
	"fmt"

	"github.com/sergi/go-diff/diffmatchpatch"
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
	Diffs []diffmatchpatch.Diff
}

func (err DiffError) Error() string {
	return "output differs"
}
