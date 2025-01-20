# Transcript

`transcript` is a CLI tool for snapshot testing other CLI tools.

The snapshot files -- called "transcripts" -- are designed to be easily human-readable
and reasonably human-writable, without sacrificing precise assertions about the behavior of
the tool under test. In practice, most transcripts can be authored interactively
and maintained in an automated way.

# Usage

Automatically record a shell session or type-out a transcript file by hand,
then use the `check` command!

```bash
cat > demo.cmdt <<EOF
$ echo stdout
1 stdout

$ echo stderr 1>&2
2 stderr

# Non-zero exit codes.
$ false
? 1
EOF

transcript check ./demo.cmdt
```

## Install

```bash
go get -u github.com/deref/transcript
```

NOTE: Transcript is not Go-specific. It is simply written in Go and Go provides
a convenient distribution mechanism. If there is expressed interest, it may be
re-packaged for various additional distribution channels.

## Record

Initial authoring of tests is performed in an interactive shell.

To record an interactive session to a file, run:

```bash
transcript shell -o example.cmdt
```

The interactive shell supports standard readline behaviors and can be exited
with `^d` or `exit` like most other shells.

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
exit status expectations. Instead, the given file will be rewritten with the
newly observed results.

## Edit

While transcript files can be edited by hand, more advanced edits can be made
using an interactive update session. The experience should be familiar to users
of `git rebase --interactive`.

NOTE: Not yet implemented.

# "Command Transcript" File Format

Transcript files represent recorded shell sessions.

`.cmdt` is the recommended file extension.

This format is intended to be human-editable, but sacrifices some ease of
hand-authoring in exchange for added functionality. Users are expected to
primarily use the `transcript` tool to create and update transcripts.

## Structure

Cmdt files are line-oriented. Each line represents an instruction to the
Transcript interpreter. Each instruction begins with an opcode, followed by a
single space. The remainder of an instruction line forms arguments to the operation specified by the opcode.

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
      configured by <code>%</code> directives in a future version of
      Transcript.
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
    <p>Configures special behaviors in the Transcript interpreter.</p>
    <p>Supported directives:</p>
    <ul>
      <li><code>no-newline</code> &mdash; Indicates that the last line of the preceding output did not end with a newline character. Applies to
      either stdout or stderr based on what's on the opcode of the previous line.
      </li>
    </ul>
  </dd>
</dl>

## Whitespace Handling

Within the arguments of an operation, whitespace is significant. Opcodes are
separated from their arguments by a single space. If there is more than one
space, the additional spaces are part of the arguments. Similarly, trailing
whitespace is part of the argument as well. This allows precise recording of
the whitespace behavior of commands under test.

If the arguments to an operation are completely empty, then the space after
the opcode is optional. Such an extraneous space is discouraged, but not
disallowed because text editors should preserve trailing whitespace in .cmdt
files to support the precision mentioned above.

Conventionally, command line tools always output a '\n' after each line,
including the last line in a file. However, there are some situations where
it is important to represent the lack of a trailing newline. In this case,
the `% no-newline` directive signifies that the last line of the transcript
is terminated with a synthetic newline. That is, the recorded output did not
have a newline and the checker should strip the synthetic newline before
checking against test output.

# Go API

In addition to the `transcript` CLI, there is a Go API for users who wish to
embed `cmdt` scripts in to their existing Go test suites.

```go
import (
  _ "embed"

  "github.com/deref/transcript/cmdtest"
)

//go:embed test.cmdt
var fs embed.FS

func TestCLI(t *testing.T) {
  f, _ := fs.Open("test.cmdt")
  defer f.Close()
  cmdtest.Check(f)
}
```

NOTE: Assuming that `./test.cmdt` uses the CLI tool you are developing, you
must first build your tool and ensure it is on `PATH`.

There is also a `CheckString` function for small, inline tests. However, prefer
to use `.cmdt` files so that the `transcript` tool can assist with updates and
edits.

# Editor Support

Editor support (for syntax highlighting) is available for [several editors](./editors).
