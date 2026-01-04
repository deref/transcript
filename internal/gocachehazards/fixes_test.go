package gocachehazards

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGoTestCacheFix_EnvVarTracked(t *testing.T) {
	t.Parallel()

	mod := newTempModule(t, moduleFiles{
		cmdt: "$ echo \"$FOO\"\n1 one\n",
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

	r1 := mod.goTest(t, map[string]string{"FOO": "one"})
	if r1.exitCode != 0 {
		t.Fatalf("first run failed:\n%s", r1.output)
	}
	if r1.cached() {
		t.Fatalf("first run unexpectedly cached:\n%s", r1.output)
	}

	// Same env var: should be cacheable and usually becomes (cached).
	r2 := mod.goTest(t, map[string]string{"FOO": "one"})
	if r2.exitCode != 0 {
		t.Fatalf("second run failed:\n%s", r2.output)
	}
	if !r2.cached() {
		t.Fatalf("second run unexpectedly not cached:\n%s", r2.output)
	}

	// Different env var: must not be cached and must fail.
	r3 := mod.goTest(t, map[string]string{"FOO": "two"})
	if r3.exitCode == 0 {
		t.Fatalf("third run unexpectedly passed:\n%s", r3.output)
	}
	if r3.cached() {
		t.Fatalf("third run unexpectedly cached:\n%s", r3.output)
	}
}

func TestGoTestCacheFix_FileDepTrackedViaDepDirective(t *testing.T) {
	t.Parallel()

	mod := newTempModule(t, moduleFiles{
		extraFiles: map[string]fileSpec{
			"dep.txt": {data: "v1\n"},
		},
		cmdt: "% dep dep.txt\n$ cat dep.txt\n1 v1\n",
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

	r2 := mod.goTest(t, nil)
	if r2.exitCode != 0 {
		t.Fatalf("second run failed:\n%s", r2.output)
	}
	if !r2.cached() {
		t.Fatalf("second run unexpectedly not cached:\n%s", r2.output)
	}

	writeFile(t, filepath.Join(mod.dir, "dep.txt"), "v2\n", 0o644)

	r3 := mod.goTest(t, nil)
	if r3.exitCode == 0 {
		t.Fatalf("third run unexpectedly passed:\n%s", r3.output)
	}
	if r3.cached() {
		t.Fatalf("third run unexpectedly cached:\n%s", r3.output)
	}
}

func TestGoTestCacheFix_FileDepTrackedViaDepfile(t *testing.T) {
	t.Parallel()

	mod := newTempModule(t, moduleFiles{
		extraFiles: map[string]fileSpec{
			"dep.txt":  {data: "v1\n"},
			"deps.txt": {data: "dep.txt\n"},
		},
		cmdt: "% dep < deps.txt\n$ cat dep.txt\n1 v1\n",
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

	r2 := mod.goTest(t, nil)
	if r2.exitCode != 0 {
		t.Fatalf("second run failed:\n%s", r2.output)
	}
	if !r2.cached() {
		t.Fatalf("second run unexpectedly not cached:\n%s", r2.output)
	}

	writeFile(t, filepath.Join(mod.dir, "dep.txt"), "v2\n", 0o644)

	r3 := mod.goTest(t, nil)
	if r3.exitCode == 0 {
		t.Fatalf("third run unexpectedly passed:\n%s", r3.output)
	}
	if r3.cached() {
		t.Fatalf("third run unexpectedly cached:\n%s", r3.output)
	}
}

func TestGoTestCacheFix_ExecutableInModuleRootTrackedViaPATH(t *testing.T) {
	t.Parallel()

	mod := newTempModule(t, moduleFiles{
		extraFiles: map[string]fileSpec{
			"bin/mytool": {data: "#!/bin/sh\necho v1\n", perm: 0o755},
		},
		cmdt: "$ mytool\n1 v1\n",
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

	pathEnv := filepath.Join(mod.dir, "bin") + string(os.PathListSeparator) + os.Getenv("PATH")

	r1 := mod.goTest(t, map[string]string{"PATH": pathEnv})
	if r1.exitCode != 0 {
		t.Fatalf("first run failed:\n%s", r1.output)
	}
	if r1.cached() {
		t.Fatalf("first run unexpectedly cached:\n%s", r1.output)
	}

	r2 := mod.goTest(t, map[string]string{"PATH": pathEnv})
	if r2.exitCode != 0 {
		t.Fatalf("second run failed:\n%s", r2.output)
	}
	if !r2.cached() {
		t.Fatalf("second run unexpectedly not cached:\n%s", r2.output)
	}

	writeFile(t, filepath.Join(mod.dir, "bin", "mytool"), "#!/bin/sh\necho v2\n", 0o755)

	r3 := mod.goTest(t, map[string]string{"PATH": pathEnv})
	if r3.exitCode == 0 {
		t.Fatalf("third run unexpectedly passed:\n%s", r3.output)
	}
	if r3.cached() {
		t.Fatalf("third run unexpectedly cached:\n%s", r3.output)
	}
}
