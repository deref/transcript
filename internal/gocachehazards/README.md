# `gocachehazards`

This package contains **tests that demonstrate `go test` caching hazards** when using
`transcript` via the Go API (`cmdtest.Check`).

These tests are intentionally written to:
- run `go test` as a subprocess (twice),
- assert whether the second run is **(cached)** or not, and
- show where cached results can be **incorrect** unless transcript ensures
  relevant dependencies are visible to the **test process**.

## Why this exists

`go test`’s package test cache keys off what the **test binary** itself observes via:
`getenv`, `stat`, `open`, `chdir` (the “testlog” mechanism). Subprocesses do not
automatically contribute to that dependency set.

Transcript tests often execute subprocesses, so changes to:
- environment variables,
- input files read by subprocesses,
- or the tool-under-test binary itself,

can be masked by `go test` caching unless those deps are also “touched” by the
test process.

## Hazards demonstrated

See `hazards_test.go` for concrete repros, including:
- Env var usage not being tracked (if transcript doesn’t hit `os.LookupEnv`).
- Subprocess file dependencies not being tracked (child reads aren’t observed).
- Deps outside the module root being ignored by `go test` cache.
- `open`-based tracking disabling caching for “too new” files (mtime cutoff).

## Running

From the repo root:

```bash
go test ./internal/gocachehazards -count=1
```

