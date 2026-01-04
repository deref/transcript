# Go `test` Cache Friendliness Plan (Remaining Work)

All core hazards in this plan are now either fixed or documented with tests.
This file tracks only the remaining follow-ups.

## Remaining

- Optional: warn (non-fatal) when a declared `% dep` file path is outside the
  module root, since `go test` will ignore it for caching.
  - Decide warning surface area: CLI only, Go API only (`cmdtest.Check`), or
    both.
  - Decide how to find module root cheaply (likely: walk upward from the
    initial working directory to `go.mod`, then treat that as the root for the
    lifetime of the transcript run).

- Doc nuance: `% dep < deps.txt` necessarily **opens** `deps.txt` due to `<`.
  If users generate the depfile during the test run, it may trip Go's
  "open-too-new" rule and prevent caching. Recommend keeping depfiles stable
  (checked in, or generated ahead of the test run).

## Follow-up: Iterating On Data Files

False cache hits are most likely when the tool binary is unchanged but input
data files change. `% dep` solves this when used correctly, but it is easy to
forget and hard to apply to indirect inputs (scripts, wrappers, dynamic paths).

Follow-up work to explore:

- Add more cache-focused tests demonstrating false cache hits when inputs are
  passed as argv to subprocesses (e.g. `mytool config.json`) without `% dep`.
- Explore an opt-in heuristic to auto-declare dependencies for "obvious input
  paths" in command arguments, with explicit tradeoffs:
  - Avoid killing cache hit rate by accidentally tracking outputs or unstable
    files.
  - Prefer conservative rules (path-like strings, existing regular files, and
    module-root filtering).
  - TDD the cache behavior to prove the heuristic reduces false cache hits
    without eliminating cache hits in common workflows.
- Evaluate input vs output treatment in the heuristic. Including output files
  as deps likely reduces cache hits, but can also be desirable when you want
  cache reuse only when outputs are stable and reproducible.
- Consider supporting common flag conventions (e.g. `--flag=value`, `-o out`,
  `--output out`) and documenting exactly what is and isn't interpreted as a
  dependency.
