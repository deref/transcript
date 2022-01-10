# Transcript

`transcript` is a CLI tool for testing other CLI tools.

TODO: Video here.

# Usage

Don't yet! You found this repository too early :)

## Record

Initial authoring of tests is performed in an interactive shell.

```bash
transcript shell -o example.cmdt
```

## Check

To interpret a transcript file and validate that the results (stdio output and
exit codes) have not changed, run the following:

```bash
transcript check example.cmdt
```

Check returns a non-zero exit code if any check failures or other errors occur.

## Update

When the CLI tools under test are modified, the quickest way to update test
files is to use the automated `update` process:

```bash
transcript update example.cmdt
```

This will interpret a command transcript file, but does not check any output or
exit status expectations. Instead, updates the given file with the newly
observed results.

NOTE: Not yet implemented.

## Edit

While transcript files can be edited by hand, more advanced edits can be made
using an interactive update session. The experience should be familiar to users
of `git rebase --interactive`.

Not yet implemented.

# "Command Transcript" File Format

Transcript files represent recorded shell sessions.

`.cmdt` is the recommended file extension.

This format is intended to be human-editable, but sacrifices some ease of
hand-authoring in exchange for added functionality. Users are expected to
primarily use the `transcript` tool to create and update transcripts.

## Structure

Cmdt files are line-oriented. Each line represents an instruction to the
Transcript interpreter. Each instruction begins with an opcode, followed by a
space. The remainder of an instruction line forms arguments to the operation
specified by the opcode.

## Operations

Operations with the following opcodes are supported:

<dl>
  <dt><code>#</code> &mdash; comment</dt>
  <dd>
    <p>
      Comments may appear anywhere in a <code>.cmdt</code> file and are ignored
      by the interpreter.
    </p>
    <p>A space is not required after the <code>#</code> opcode.</p>
    <p>Blank lines are also treated as comments.</p>
  </dd>

  <dt><code>$</code> &mdash; command</dt>
  <dd>
    <p>Run a shell command.</p>
    <p>
      Supports the subset of Bash syntax provided by
      <a href="https://github.com/mvdan/sh#gosh">mvdan/sh</a>.
    </p>
  </dd>

  <dt><code>1</code>, <code>2</code> &mdash; output</dt>
  <dd>
    <p>
      Match a line of output from a particular stdio stream of the previously
      run command.
    </p>
    <p>
      The opcodes are named after the file descriptors of stdout
      (<code>1</code>) and stderr (<code>2</code>) respectively.
    </p>
    <p>
      Output lines are matched exactly. More flexible matching may be
      configured by `%` directives in a future version of Transcript.
    </p>
    <p>
      Transcript checking assumes that the interleaving of stdout and stderr
      lines is significant and that output lines are written atomically.
      The ordering of concurrent writes to both streams is undefined, which
      will lead to flakey tests. Incrementally written lines will be buffered,
      which may mask text interleaving issues that would affect users. Both of
      these shortcomings may be mitigated in the future.
    </p>
  </dd>

  <dt><code>?</code> &mdash; exit-code</dt>
  <dd>
    <p>Exit code of the previously run command.</p>
    <p>If omitted, the exit code defaults to <code>0</code>.</p>
  </dd>

  <dt><code>%</code> &mdash; directive</dt>
  <dd>
    <p>Reserved for future use by Transcript.</p>
  </dd>
</dl>

# Go API

In addition to the `transcript` CLI, there is a Go API for users who wish to
embed `cmdt` scripts in to their existing Go test suites.

```go
import (
  _ "embed"

  "github.com/deref/transcript/cmdtest"
)

////go:embed test.cmdt
var fs embed.FS

func TestCLI(t *testing.T) {
  f, _ := fs.Open("test.cmdt")
  defer f.Close()
  cmdtest.Check(f)
}
```

There are also `CheckString` function for small, inline tests. However, prefer
to use `.cmdt` files so that the `transcript` tool can assist with updates and
edits.
