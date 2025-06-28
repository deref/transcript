#!/bin/bash

# This file exercises the check CLI.
# It's intentionally redundant with transcript_test.go, which exercises the Go API.

set -e

export PATH="$PWD/bin:$PATH"
if [[ $(which transcript) != "$PWD/bin/transcript" ]]; then
  echo 'built transcript not on PATH' 1>&2
  exit 1
fi

./build.sh

set -x

# Run transcript checks from within the tests directory
# All the "*.cmdt" files are used directly as tests.
# The "*.fail" tests are run indirectly by 'meta.cmdt'.
(
  cd tests
  transcript check *.cmdt
)

go test -v ./...
