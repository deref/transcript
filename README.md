# Transcript

`transcript` is a CLI tool for snapshot testing command-line programs.

Tests are written in a small, line-oriented format (usually `*.cmdt`) that
captures a shell session: commands, stdout/stderr, and exit codes. The format
is designed to be readable in reviews and easy to update when outputs change.

## Highlights

- Record sessions interactively (`transcript shell`)
- Check transcripts in CI (`transcript check`)
- Update expectations when outputs change (`transcript update`)
- Reference external files for large or binary output (`1<` / `2<`)
- Embed transcripts in Go tests via `cmdtest.Check`

## Quick Start

Create a transcript and check it:

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
go install github.com/deref/transcript@latest
```

Transcript is not Go-specific, but it is written in Go and can be used in Go
test suites via `cmdtest.Check`.

## Documentation

- [User guide](docs/user-guide.md)
- [File format reference](docs/reference.md)
- [Go test caching notes](docs/test-cache.md)
- [Editor support](editors/)
