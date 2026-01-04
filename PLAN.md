# Go `test` Cache Friendliness Plan

This repository provides:
- A `transcript` CLI for snapshot-testing other CLIs (useful beyond Go).
- A Go API (`cmdtest.Check`) for embedding `.cmdt` scripts into Go tests.

One motivation for the Go API is integration with `go test`’s **package test cache**.
However, there are important holes: `go test` only keys cached results off what the
**test process** itself does (via the internal “testlog”: `getenv`, `stat`, `open`, `chdir`),
not what subprocesses do.

This plan enumerates the caching hazards we want to **reproduce with failing tests (TDD)**,
then fix one-by-one, and finally add an ergonomic dependency declaration mechanism (depfile,
directives) that keeps test logic in `.cmdt`.

## Goals

- Make `cmdtest.Check` “cache-correct” where possible:
  - If a transcript depends on a file/env var *within the module root*, changing it should
    invalidate `go test` cache.
- Keep test logic in `.cmdt` files (minimal Go glue).
- Keep `cmdtest` API growth small.
- Keep the CLI workflow intact for non-Go projects.
- Document remaining limits clearly (some are fundamental to `go test` caching).

## Non-Goals / Constraints

- We cannot automatically track every dependency used by subprocesses.
- Dependencies outside the module root are ignored by `go test` caching (by design).
- We won’t require network access or special tooling in tests.

## How `go test` caching “sees” dependencies (important constraint)

`go test` caches passing package test results in package-list mode (e.g. `go test .`, `go test ./...`).
When computing the cache key it considers:
- the test binary + cacheable flags, and
- files and env vars observed by the test binary itself via the “testlog” actions:
  `getenv`, `stat`, `open`, `chdir`.

That means **subprocess reads do not count** unless the test process also `stat`/`open`s the same inputs.

## Hazards to TDD and Fix (in order)

Each hazard has:
- **Why it happens**
- **TDD**: a failing integration test demonstrating the bad caching behavior
- **Fix**: the change we’ll make to transcript
- **Docs**: what we recommend users do

### H1: Environment variables referenced in `.cmdt` aren’t tracked by `go test` cache

**Why:** transcript’s shell runner currently uses a snapshot environment (a copy of `os.Environ()`),
so `$FOO` expansions can avoid calling `os.LookupEnv`. If the test process never calls
`os.LookupEnv("FOO")`, `go test` won’t record that env var in its cache key.

**TDD (integration):**
- Create a temp Go module with a test that runs `cmdtest.Check` on an embedded `.cmdt`.
- `.cmdt` prints an env var (e.g. `$ printf "%s" "$FOO"`).
- Run `go test .` with `FOO=one` (expect pass, not cached).
- Run `go test .` again with `FOO=two` (expect **not cached** and **fail**).
- Current bug will often be: second run is `(cached)` and still passes.

**Fix:**
- Plumb environment lookups through `os.LookupEnv` (or `os.Getenv`) so they are logged
  as `getenv` actions by the test process during transcript execution.

**Docs:**
- After fix: recommend using env vars normally in `.cmdt`; caching will key on them.

### H2: File dependencies used by subprocesses are invisible to `go test` cache

**Why:** `.cmdt` typically runs external tools. If a tool reads `dep.txt`, that read happens in
the child process, so `go test` doesn’t see it.

**TDD (integration):**
- `.cmdt` runs a subprocess that reads `dep.txt` and prints its contents (e.g. `cat dep.txt`).
- The Go test process does *not* otherwise read/stat `dep.txt`.
- Run `go test .` (pass), modify `dep.txt`, rerun `go test .`.
- Demonstrate that the second run can be `(cached)` even though the output should change.

**Fix (core):**
- We can’t make child reads visible automatically, but we can provide primitives so the test
  process “touches” declared deps (via `stat`, not `open`) before running commands.

**Docs (short-term mitigation without new directives):**
- Users can force a `stat` in-process using a shell builtin before the real command:
  - `test -r dep.txt` (builtin `test` / `[`) records a `stat` and doesn’t mutate the file.
- Avoid `touch` (mutates) and avoid patterns that force `open` (can disable caching for “too new” files).

### H3: Working-directory semantics can desync dependency resolution

**Why:** `.cmdt` supports `cd` that persists across commands in the transcript shell runner.
But transcript also reads “expected file output” referenced by `1< file` / `2< file`.
If those file paths are resolved against the Go process’s CWD instead of the transcript runner’s CWD,
we get both correctness bugs and cache-tracking bugs.

**TDD (unit or integration):**
- Create a `.cmdt` that `cd`s into a subdir and then references an expected file via `1< expected.txt`
  (or otherwise relies on relative paths post-`cd`).
- Assert the check reads the correct file and fails when it changes.

**Fix:**
- Make file-reference resolution (`1<`, `2<`) use the transcript runner’s current directory at that point.
- Ensure the “touching” for cache tracking uses absolute paths within the module root.

**Docs:**
- Relative paths in `.cmdt` are relative to the transcript session’s working directory (post-`cd`).

### H4: Tool-under-test binary changes aren’t tracked (PATH/lookup hazard)

**Why:** Transcripts commonly run `mytool` found on `PATH`. Rebuilding the binary doesn’t necessarily
invalidate cache unless the test process also observes that binary file (via `stat`/`open`) within module root.

**TDD (integration):**
- In a temp module, create a `bin/mytool` file (or small Go-built tool) whose output changes.
- `.cmdt` runs `bin/mytool` (or `mytool` with `PATH` pointing at `./bin`).
- Demonstrate cache can remain `(cached)` across tool changes unless the test process tracks it.

**Fix:**
- Provide a first-class way to declare “executable deps” (implemented as a `stat` of the resolved path),
  or document a depfile entry pattern so the test process always `stat`s the tool binary.

**Docs:**
- Recommend keeping the tool binary under the module root (e.g. `./bin/mytool`) and declaring it as a dep.

### H5: Deps outside module root never affect `go test` cache

**Why:** `go test` intentionally ignores `open`/`stat` entries outside module root (or GOPATH/GOROOT).

**TDD (integration):**
- Demonstrate that a dep in `/tmp` (or other outside-root location) does not invalidate cache.

**Fix:**
- No fix (upstream behavior). We should detect and warn (optional) when a declared dep is outside root.

**Docs:**
- Call out: deps outside module root won’t make `go test` rerun the package.

### H6: `open`-based dependency tracking can disable caching for “too new” files

**Why:** `go test` uses a “modTime cutoff” for opened regular files; if a file is very recent it can refuse to cache.

**TDD (optional):**
- Show that using an `open` pattern on a freshly-written file can lead to non-cached runs.

**Fix / Guidance:**
- Prefer `stat`-based tracking for declared deps.

## Dependency Declaration Design (after H1–H3 fixes)

We want `.cmdt` to remain the source of truth, but we also want a way to:
- declare a set of file/env/exe deps once,
- reuse across many commands (including scripts), and
- keep command transcripts readable.

Proposed direction: a **depfile** mechanism, with optional `.cmdt` sugar.

### Depfile (proposal)

- New directive: `% depfile path/to/deps.txt`
- Depfile format (candidate): line-based entries:
  - `file relative/or/absolute/path`
  - `env VAR_NAME`
  - `exe relative/or/absolute/path` (or `exe PATH_LOOKUP_NAME`)
  - blank lines and `# comments` ignored
- Semantics:
  - Each declared dep causes the test process to `stat` (and optionally `getenv`) before executing commands.
  - Prefer `stat` over `open` for stable caching.
  - Paths are resolved relative to the transcript session’s current working directory at the point the depfile is declared.

### Optional `.cmdt` sugar (proposal)

- `% dep file path`
- `% dep env VAR`
- `% dep exe path-or-name`

These can expand to the same internal “touch” mechanism as depfile, but depfile is the primary answer
to “can’t abstract over it”.

## Test Harness Strategy

To actually observe `go test` caching, tests must run `go test` as a subprocess at least twice and:
- capture stdout, and
- assert on `ok ... (cached)` vs non-cached output.

We should build a small helper in tests to:
- create a temp module (`go mod init`, write files),
- run `go test .` in that module with controlled env,
- mutate deps between runs,
- assert cached/non-cached and pass/fail.

## Milestones

1. Add `PLAN.md` (this file) + a short doc section in `README.md` about caching hazards and recommended patterns.
2. Implement and land H1 fix (env tracking) with an integration test.
3. Implement and land H3 fix (cwd-correct file refs / tracking) with tests.
4. Add integration tests for H2/H4 showing hazards and recommended mitigations.
5. Design and implement depfile (+ optional directives), with tests demonstrating that declared deps invalidate cache.
6. Update docs with the recommended workflow:
   - keep deps under module root,
   - prefer `stat`-style tracking,
   - use depfile for larger suites.

