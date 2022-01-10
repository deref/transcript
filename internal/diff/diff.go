package diff

import (
	"strings"

	"github.com/pmezard/go-difflib/difflib"
	"github.com/sergi/go-diff/diffmatchpatch"
)

// Returns newline-terminated diff with CLI color ANSI codes.
func Color(expected, actual string) string {
	dmp := diffmatchpatch.New()
	diffs := dmp.DiffMain(expected, actual, true)
	return diffmatchpatch.New().DiffPrettyText(diffs)
}

// Returns newline-terminated, unified plain-text diff.
func Plain(expected, actual string) string {
	diff := difflib.UnifiedDiff{
		A:        difflib.SplitLines(expected),
		B:        difflib.SplitLines(actual),
		FromFile: "expected",
		ToFile:   "actual",
		Context:  3,
	}
	text, err := difflib.GetUnifiedDiffString(diff)
	if err != nil {
		panic(err)
	}
	return strings.TrimSpace(text)
}
