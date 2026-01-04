package gocachehazards

// This test package demonstrates `go test` caching hazards relevant to transcript.
//
// `go test`’s package test cache keys off what the *test process* itself does
// (via the internal testlog actions: getenv/stat/open/chdir). Dependencies used
// only by subprocesses do not automatically influence the cache key.
//
// The tests here run `go test` as a subprocess twice and assert whether the
// second run is (cached). Many of these hazards show up as an *incorrect cached
// pass* after changing a dependency that the transcript relies on.

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

type goTestResult struct {
	output   string
	exitCode int
}

func (r goTestResult) cached() bool {
	return strings.Contains(r.output, "(cached)")
}

func TestGoTestCacheHazard_EnvVarNotTracked(t *testing.T) {
	t.Parallel()

	mod := newTempModule(t, moduleFiles{
		cmdt: strings.TrimSpace(`
$ echo "$FOO"
1 one
`) + "\n",
		testGo: `
package main

import (
  _ "embed"
  "testing"
  "github.com/deref/transcript/cmdtest"
)

//go:embed test.cmdt
var cmdt string

func TestTranscript(t *testing.T) {
  cmdtest.CheckString(t, cmdt)
}
`,
	})

	r1 := mod.goTest(t, map[string]string{
		"FOO": "one",
	})
	if r1.exitCode != 0 {
		t.Fatalf("first run failed:\n%s", r1.output)
	}
	if r1.cached() {
		t.Fatalf("first run unexpectedly cached:\n%s", r1.output)
	}

	r2 := mod.goTest(t, map[string]string{
		"FOO": "two",
	})
	if r2.exitCode != 0 || !r2.cached() {
		t.Skipf("hazard not reproduced (likely fixed):\n%s", r2.output)
	}
}

func TestGoTestCacheHazard_SubprocessFileDependencyNotTracked(t *testing.T) {
	t.Parallel()

	mod := newTempModule(t, moduleFiles{
		extraFiles: map[string]fileSpec{
			"dep.txt": {data: "v1\n"},
		},
		cmdt: strings.TrimSpace(`
$ cat dep.txt
1 v1
`) + "\n",
		testGo: `
package main

import (
  _ "embed"
  "testing"
  "github.com/deref/transcript/cmdtest"
)

//go:embed test.cmdt
var cmdt string

func TestTranscript(t *testing.T) {
  cmdtest.CheckString(t, cmdt)
}
`,
	})

	r1 := mod.goTest(t, nil)
	if r1.exitCode != 0 {
		t.Fatalf("first run failed:\n%s", r1.output)
	}
	if r1.cached() {
		t.Fatalf("first run unexpectedly cached:\n%s", r1.output)
	}

	writeFile(t, filepath.Join(mod.dir, "dep.txt"), "v2\n", 0o644)

	r2 := mod.goTest(t, nil)
	if r2.exitCode != 0 {
		t.Fatalf("second run failed (wanted cached pass to demonstrate hazard):\n%s", r2.output)
	}
	if !r2.cached() {
		t.Fatalf("second run unexpectedly not cached (hazard not reproduced):\n%s", r2.output)
	}
}

func TestGoTestCacheHazard_DepsOutsideModuleRootIgnored(t *testing.T) {
	t.Parallel()

	mod := newTempModule(t, moduleFiles{
		cmdt: strings.TrimSpace(`
$ test -e ../external.txt
$ cat ../external.txt
1 v1
`) + "\n",
		testGo: `
package main

import (
  _ "embed"
  "testing"
  "github.com/deref/transcript/cmdtest"
)

//go:embed test.cmdt
var cmdt string

func TestTranscript(t *testing.T) {
  cmdtest.CheckString(t, cmdt)
}
`,
	})

	externalPath := filepath.Join(filepath.Dir(mod.dir), "external.txt")
	writeFile(t, externalPath, "v1\n", 0o644)

	r1 := mod.goTest(t, nil)
	if r1.exitCode != 0 {
		t.Fatalf("first run failed:\n%s", r1.output)
	}
	if r1.cached() {
		t.Fatalf("first run unexpectedly cached:\n%s", r1.output)
	}

	writeFile(t, externalPath, "v2\n", 0o644)

	r2 := mod.goTest(t, nil)
	if r2.exitCode != 0 {
		t.Fatalf("second run failed (wanted cached pass to demonstrate hazard):\n%s", r2.output)
	}
	if !r2.cached() {
		t.Fatalf("second run unexpectedly not cached (hazard not reproduced):\n%s", r2.output)
	}
}

func TestGoTestCacheHazard_OpenTooNewPreventsCaching(t *testing.T) {
	t.Parallel()

	mod := newTempModule(t, moduleFiles{
		cmdt: strings.TrimSpace(`
$ echo hi > dep.txt
$ cat < dep.txt
1 hi
`) + "\n",
		testGo: `
package main

import (
  _ "embed"
  "testing"
  "github.com/deref/transcript/cmdtest"
)

//go:embed test.cmdt
var cmdt string

func TestTranscript(t *testing.T) {
  cmdtest.CheckString(t, cmdt)
}
`,
	})

	r1 := mod.goTest(t, nil)
	if r1.exitCode != 0 {
		t.Fatalf("first run failed:\n%s", r1.output)
	}

	// The transcript opens a freshly-written file via redirection ("< dep.txt").
	// `go test` intentionally refuses to cache runs that "open" very new files.
	r2 := mod.goTest(t, nil)
	if r2.exitCode != 0 {
		t.Fatalf("second run failed:\n%s", r2.output)
	}
	if r2.cached() {
		t.Fatalf("second run unexpectedly cached despite open-too-new pattern:\n%s", r2.output)
	}
}

func TestGoTestCacheHazard_ExecutableOutsideModuleRootNotTracked(t *testing.T) {
	t.Parallel()

	mod := newTempModule(t, moduleFiles{
		cmdt: strings.TrimSpace(`
$ mytool
1 v1
`) + "\n",
		testGo: `
package main

import (
  _ "embed"
  "testing"
  "github.com/deref/transcript/cmdtest"
)

//go:embed test.cmdt
var cmdt string

func TestTranscript(t *testing.T) {
  cmdtest.CheckString(t, cmdt)
}
`,
	})

	// Put an executable on PATH outside the module root.
	externalBinDir := filepath.Join(filepath.Dir(mod.dir), "external-bin")
	if err := os.MkdirAll(externalBinDir, 0o755); err != nil {
		t.Fatalf("mkdir external bin: %v", err)
	}
	mytool := filepath.Join(externalBinDir, "mytool")
	writeFile(t, mytool, "#!/bin/sh\necho v1\n", 0o755)

	pathEnv := externalBinDir + string(os.PathListSeparator) + os.Getenv("PATH")

	r1 := mod.goTest(t, map[string]string{
		"PATH": pathEnv,
	})
	if r1.exitCode != 0 {
		t.Fatalf("first run failed:\n%s", r1.output)
	}
	if r1.cached() {
		t.Fatalf("first run unexpectedly cached:\n%s", r1.output)
	}

	writeFile(t, mytool, "#!/bin/sh\necho v2\n", 0o755)

	r2 := mod.goTest(t, map[string]string{
		"PATH": pathEnv,
	})
	if r2.exitCode != 0 || !r2.cached() {
		t.Skipf("hazard not reproduced (possibly non-cached due to local go test behavior):\n%s", r2.output)
	}
}

type fileSpec struct {
	data string
	perm os.FileMode
}

type moduleFiles struct {
	cmdt       string
	testGo     string
	extraFiles map[string]fileSpec
}

type tempModule struct {
	dir       string
	gocache   string
	repoRoot  string
	modPath   string
	goVersion string
}

func newTempModule(t *testing.T, files moduleFiles) *tempModule {
	t.Helper()

	pkgDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	repoRoot := findRepoRoot(t, pkgDir)

	baseDir := t.TempDir()
	dir := filepath.Join(baseDir, "mod")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir mod: %v", err)
	}

	gocache := filepath.Join(baseDir, "gocache")
	if err := os.MkdirAll(gocache, 0o755); err != nil {
		t.Fatalf("mkdir gocache: %v", err)
	}

	modPath := fmt.Sprintf("example.com/transcript-gocachehazard/%d-%d", os.Getpid(), time.Now().UnixNano())

	writeFile(t, filepath.Join(dir, "go.mod"), fmt.Sprintf(strings.TrimSpace(`
module %s

go 1.24

require github.com/deref/transcript v0.0.0

replace github.com/deref/transcript => %s
`)+"\n", modPath, filepath.ToSlash(repoRoot)), 0o644)

	// Ensure the temp module has sums for transcript's dependencies so it can
	// build fully offline.
	copyFile(t, filepath.Join(repoRoot, "go.sum"), filepath.Join(dir, "go.sum"))

	writeFile(t, filepath.Join(dir, "main_test.go"), files.testGo, 0o644)
	writeFile(t, filepath.Join(dir, "test.cmdt"), files.cmdt, 0o644)

	for rel, spec := range files.extraFiles {
		perm := spec.perm
		if perm == 0 {
			perm = 0o644
		}
		writeFile(t, filepath.Join(dir, filepath.FromSlash(rel)), spec.data, perm)
	}

	return &tempModule{
		dir:      dir,
		gocache:  gocache,
		repoRoot: repoRoot,
		modPath:  modPath,
	}
}

func (m *tempModule) goTest(t *testing.T, extraEnv map[string]string) goTestResult {
	t.Helper()

	env := append([]string{}, os.Environ()...)
	env = append(env,
		"GOWORK=off",
		"GOFLAGS=",
		"GOPROXY=off",
		"GOSUMDB=off",
		"GODEBUG=",
		"GOCACHE="+m.gocache,
	)
	for k, v := range extraEnv {
		env = append(env, k+"="+v)
	}

	cmd := exec.Command("go", "test", "-mod=mod", "-run", "TestTranscript$", ".")
	cmd.Dir = m.dir
	cmd.Env = env
	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf
	err := cmd.Run()
	exitCode := 0
	if err != nil {
		var ee *exec.ExitError
		if !errors.As(err, &ee) {
			t.Fatalf("go test run error: %v\noutput:\n%s", err, buf.String())
		}
		exitCode = ee.ExitCode()
	}

	return goTestResult{
		output:   buf.String(),
		exitCode: exitCode,
	}
}

func writeFile(t *testing.T, path, data string, perm os.FileMode) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdirall %s: %v", path, err)
	}
	if err := os.WriteFile(path, []byte(data), perm); err != nil {
		t.Fatalf("writefile %s: %v", path, err)
	}
}

func copyFile(t *testing.T, src, dst string) {
	t.Helper()
	b, err := os.ReadFile(src)
	if err != nil {
		t.Fatalf("readfile %s: %v", src, err)
	}
	writeFile(t, dst, string(b), 0o644)
}

func findRepoRoot(t *testing.T, start string) string {
	t.Helper()
	dir := start
	for {
		gomod := filepath.Join(dir, "go.mod")
		if b, err := os.ReadFile(gomod); err == nil {
			if strings.Contains(string(b), "module github.com/deref/transcript") {
				return dir
			}
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatalf("could not find repo root from %s", start)
		}
		dir = parent
	}
}
