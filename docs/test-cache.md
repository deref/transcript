# Go `test` Caching Notes

When `cmdtest.Check` runs inside `go test`, successful package test results may
be reused from the Go package test cache.

The key constraint: `go test` only keys cached results off what the test
process itself observes (via internal "testlog" actions like `getenv`, `stat`,
`open`, and `chdir`). Dependencies used only by subprocesses do not
automatically influence the cache key.

The `internal/gocachehazards` package contains end-to-end tests demonstrating
these behaviors.

## Goals And Tradeoffs

There are two competing goals when using transcript under `go test` caching:

- Avoid false cache hits (a test is reused from cache even though an input
  changed), since that means missed test coverage.
- Preserve cache hits when possible, since it speeds up iteration.

In practice, the tool-under-test binary often changes frequently, which busts
the cache and makes the second goal less important. False cache hits are most
likely when iterating on data files (configs, fixtures) without changing the
tool binary.

## What Transcript Does

Transcript tries to make common dependencies visible to the test process:

- Environment variable expansions in the transcript shell consult `os.LookupEnv`
  so `go test` can record `getenv` actions.
- Input redirections like `< config.json` are opened by the transcript shell in
  the test process, so `go test` can record `open` actions for them.
- `% dep` causes the test process to `stat` file dependencies and `getenv` env
  dependencies, so changes invalidate cached results.
- `% dep` uses `stat` rather than `open` for file deps to avoid Go's "open too
  new" caching cutoff for freshly modified files.

## Using `% dep`

Declare a file dependency:

```text
% dep config.json
```

Declare an environment variable dependency:

```text
% dep '$PATH'
```

Declare a set of dependencies from a depfile:

```text
% dep < deps.txt
```

Depfiles are line-oriented data files (no shell expansion). See
`docs/reference.md` for the depfile format.

### Avoid Generating A Depfile In The Transcript

It's tempting to generate a depfile during a transcript run and then declare it
with `% dep < ...`, but don't do this if you care about `go test` caching.

`% dep < deps.txt` necessarily opens `deps.txt` (because of the `<`
redirection).
If the depfile was just written or modified, Go's "open too new" cutoff can
refuse to cache the package test result.

Prefer stable depfiles (checked in, or otherwise not rewritten on every test
run) when caching matters.

## Best Practices

- Prefer explicit dependencies. `% dep` is the reliable way to avoid false cache
  hits when subprocesses read inputs.
- Declare subprocess-read inputs:
  - `% dep config.json`
  - `% dep < deps.txt` (depfile)
- Keep dependencies under the module root. `go test` intentionally ignores file
  dependencies outside the module root when computing cache keys.
- Keep the tool-under-test binary under the module root (for example `./bin`)
  and add that directory to `PATH`.
- If you want cache hits while iterating on data files, avoid rebuilding the
  tool on every `go test` run.
- Avoid patterns that force the test process to `open` a freshly-written file
  when you care about caching. For example, if `deps.txt` is generated during
  the test run, `% dep < deps.txt` may trip Go's "open too new" rule.

## Hazards And Status

These hazards are demonstrated in `internal/gocachehazards/hazards_test.go`.

### Env var expansions

If `$FOO` expansions do not call `os.LookupEnv("FOO")`, changing `FOO` may not
invalidate the `go test` cache. Transcript fixes this by using `os.LookupEnv`
for expansions in the shell runner.

### Subprocess file reads

If a subprocess reads `dep.txt`, `go test` will not automatically notice,
regardless of whether `dep.txt` is passed as an argument or discovered
indirectly (via env vars, config files, wrapper scripts, etc).

Declare subprocess inputs with `% dep dep.txt` so the test process `stat`s the
file.

### Tool binaries found via `PATH`

If your tool binary lives outside the module root (for example
`/usr/local/bin`), changes to the tool may not invalidate the `go test` cache.
Keep the tool binary under the module root and reference it via `PATH`.

### Deps outside module root

`go test` intentionally ignores file deps outside the module root for caching.
This cannot be fixed in transcript; keep deps under the module root.

### "Open too new"

If the test process `open`s a very recently modified regular file, `go test`
can refuse to cache the run. `% dep` uses `stat` for file deps, but `% dep <
deps.txt` still opens the depfile due to `<`. Prefer stable depfiles.

## Common Hazards (No Fix)

- Deps outside module root are ignored by `go test` caching.
- Executables found on `PATH` outside module root are ignored by `go test`
  caching.
- If the test process `open`s a very recently modified file, `go test` can
  refuse to cache the run.
