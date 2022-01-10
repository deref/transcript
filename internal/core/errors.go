package core

import (
	"fmt"

	"github.com/deref/transcript/internal/diff"
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
	Expected string
	Actual   string
}

func (err DiffError) Error() string {
	return "output differs"
}

func (err DiffError) Color() string {
	return diff.Color(err.Expected, err.Actual)
}

func (err DiffError) Plain() string {
	return diff.Plain(err.Expected, err.Actual)
}
