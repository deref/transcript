package core

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Hazard (H3 in PLAN.md): after a transcript `cd`, expected file references
// (`1< file` / `2< file`) should resolve relative to the transcript session's
// working directory, not the Go process working directory.
//
// Beyond correctness, this also matters for `go test` caching: if transcript
// reads the wrong file path (or doesn't stat/open the intended path), the cache
// key will be incorrect.
func TestChecker_Hazard_FileOutputIgnoresTranscriptWorkingDir(t *testing.T) {
	dir, err := os.MkdirTemp(".", "transcript-cwd-hazard-*")
	if err != nil {
		t.Fatalf("mkdtemp: %v", err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(dir) })

	subdir := filepath.Join(dir, "sub")
	if err := os.MkdirAll(subdir, 0o755); err != nil {
		t.Fatalf("mkdir sub: %v", err)
	}

	// Expected output file exists only in the subdir.
	if err := os.WriteFile(filepath.Join(subdir, "expected.txt"), []byte("ok\n"), 0o644); err != nil {
		t.Fatalf("write expected: %v", err)
	}

	originalWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(originalWD) })

	if err := os.Chdir(dir); err != nil {
		t.Fatalf("chdir: %v", err)
	}

	// The command runs in "sub" (via shell builtin "cd") and prints "ok".
	// The expected file reference is relative and should also resolve in "sub".
	cmdt := strings.TrimSpace(`
$ cd sub
$ cat expected.txt
1< expected.txt
`) + "\n"

	ckr := &Checker{}
	err = ckr.CheckTranscript(context.Background(), strings.NewReader(cmdt))
	if err == nil {
		t.Fatalf("expected failure demonstrating hazard, got nil")
	}

	// The bug manifests as a read failure of "expected.txt" from the process CWD.
	if !strings.Contains(err.Error(), "reading expected file expected.txt") {
		t.Fatalf("unexpected error (wanted file-read failure): %v", err)
	}
}
