# Command Transcript (`*.cmdt`) Reference

Transcripts are line-oriented. Each line begins with an opcode, optionally
followed by a single space and the opcode's arguments.

Within the arguments of an operation, whitespace is significant. Opcodes are
separated from their arguments by a single space. If there is more than one
space, the additional spaces are part of the arguments. Trailing whitespace is
also part of the argument.

## Opcodes

### `#` comment

Comments and blank lines are ignored.

### `$` command

Run a shell command. Commands are interpreted by `mvdan.cc/sh` (a bash-like
shell in Go), so quoting, parameter expansion, and redirections generally work
as expected.

### `1` / `2` output

Match an output line from the previous command:

- `1` is stdout
- `2` is stderr

Lines are matched exactly.

Transcript uses deterministic ordering (stderr first, then stdout). True
chronological interleaving cannot be preserved due to OS pipe buffering.

### `1<` / `2<` file output

Like `1` / `2`, but reference a file containing expected output.

- For text files, the checker compares the file's contents as if the lines were
  inlined as `1 ...` / `2 ...` checks.
- For binary files, the checker expects the command output to match a file
  reference line (for example `1< 001.bin`).

File paths are interpreted relative to the transcript session's current working
directory (including after `cd` commands).

### `?` exit code

Match the exit code of the previous command. If omitted, the expected exit code
defaults to `0`.

## Directives (`% ...`)

Directives configure special interpreter behaviors.

### `% no-newline`

Indicates that the last output line did not end with a newline character. It
applies to the output stream (stdout/stderr) of the most recent output check.

This directive exists because `.cmdt` is line-based, but programs sometimes
emit a final line without a trailing newline.

### `% dep <shell-args...>`

Declares dependencies for the current transcript session, primarily for Go test
caching when using `cmdtest.Check`.

The payload is parsed and expanded by the transcript shell runner (so quoting,
parameter expansion, and `< depfile` redirection work). The expanded result is
then interpreted by an intrinsic `dep` command:

- Arguments starting with `$` declare environment variable dependencies. The
  remainder of the argument is treated as the variable name (for example
  `% dep '$PATH'` declares `PATH`).
- All other arguments declare file path dependencies.
- If stdin is redirected into `% dep` (for example `% dep < deps.txt`), the
  depfile is read and each line is interpreted as an additional dependency.

Dependency declarations are best-effort: missing files or unset env vars do not
fail the transcript check by themselves.

The `% dep` directive intentionally rejects shell constructs that could execute
other commands (like command substitution), since those would introduce hidden
subprocess dependencies that `go test` cannot reliably track.

## Depfile Format

Depfiles are line-oriented data files. Depfiles do not perform shell expansion.

- Blank lines are ignored.
- Lines beginning with `#` are comments.
- Lines beginning with `$` declare environment variable dependencies (the
  remainder of the line is the variable name).
- All other lines declare file path dependencies (the entire line is the file
  path).

Depfiles support minimal escaping:

- `\\` for a literal backslash
- `\$` for a literal leading `$` in a file path
- `\n` for a literal newline character in a path/name

## Working Directory

Transcript inherits the working directory from the process that launches it.
Directory changes (such as `cd`) persist throughout the transcript session.

## Binary Output

Transcript detects binary output using heuristics.

- Text output is recorded inline using `1` / `2`.
- Binary output is written to numbered files (`001.bin`, `002.bin`, ...), and
  referenced via `1<` / `2<`.

This applies to both interactive recording (`transcript shell`) and automatic
updates (`transcript update`).
