# User Guide

This guide covers common workflows for writing and maintaining `*.cmdt` tests.

For the full file format and all opcodes, see `docs/reference.md`.

## Install

```bash
go install github.com/deref/transcript@latest
```

## Your First Transcript

Create a file:

```bash
cat > demo.cmdt <<EOF
$ echo hello
1 hello
EOF
```

Check it:

```bash
transcript check demo.cmdt
```

## Example: Snapshot A Tool's Help Output

```bash
cat > help.cmdt <<EOF
$ mytool --help
1 Usage:
1   mytool [flags] <args...>
EOF

transcript check help.cmdt
```

## Record (Interactive)

To author tests quickly, record an interactive shell session:

```bash
transcript shell -o example.cmdt
```

Exit the shell with `Ctrl-D` or `exit`.

## Update

When outputs change, regenerate expectations:

```bash
transcript update example.cmdt
```

`update` re-runs commands and rewrites the transcript with newly observed
stdout/stderr and exit codes.

## Working With Files

If output is large, or if the output is binary, transcripts can reference an
external file via `1<`/`2<`. `transcript update` will automatically create
numbered `*.bin` files for binary output.

## Go Tests

You can embed `*.cmdt` scripts in Go tests:

```go
import (
  _ "embed"
  "testing"

  "github.com/deref/transcript/cmdtest"
)

//go:embed test.cmdt
var cmdt string

func TestCLI(t *testing.T) {
  cmdtest.CheckString(t, cmdt)
}
```

Your transcript typically runs the tool-under-test via `PATH`, so ensure your
test setup builds the tool and places it on `PATH` before running `go test`.

## Go Test Caching And Dependencies

When you run transcripts via `cmdtest.Check` inside `go test`, the package test
cache only considers dependencies observed by the test process itself. If your
transcript runs subprocesses that read inputs, declare those inputs using
`% dep` so changes invalidate the cache.

Input redirections like `< config.json` are already opened by the transcript
shell in the test process, so they are typically visible to `go test` caching
without `% dep`.

Example:

```text
% dep config.json
% dep '$PATH'
% dep < deps.txt

$ mytool --config config.json
1 ok
```

Note: `% dep < deps.txt` necessarily opens `deps.txt` in the test process. If
the depfile is generated or modified during the `go test` run (or is otherwise
freshly written), Go may refuse to cache the package test result. Prefer stable
depfiles when caching matters.

### Practical Guidance

- If your test setup rebuilds the tool-under-test on every `go test` run, Go's
  package test cache will usually be invalidated anyway. That is often fine and
  can be a good default during active development.
- If you are iterating on data files (configs, fixtures, golden inputs) without
  changing the tool binary, `go test` caching becomes more relevant. In that
  context, `% dep` is important to avoid false cache hits.

See `docs/test-cache.md` for details, tradeoffs, and remaining hazards.

## Editor Support

This repo includes syntax highlighting for `*.cmdt` files:

- Vim: `editors/vim/`
- VS Code: `editors/vscode/`

In general, treat `*.cmdt` as a first-class source file:

- Use a fixed-width font and preserve whitespace.
- Prefer editing `*.cmdt` over embedding transcripts as Go string literals.
