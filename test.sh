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
# All test.cmdt files in subdirectories are the actual tests.
# The "*.fail" tests are run indirectly by 'meta.cmdt'.
(
  cd tests
  # Find all test.cmdt files and run each from its own directory
  find . -name "test.cmdt" | sort | while read testfile; do
    testdir=$(dirname "$testfile")
    echo "Running test in $testdir"
    (cd "$testdir" && transcript check test.cmdt)
  done
)

go test -v ./...
