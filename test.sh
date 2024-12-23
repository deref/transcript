#!/bin/bash

set -e

export PATH="$PWD:$PATH"

# All the "*.cmdt" files are used directly as tests.
# The "*.fail" tests are run indirectly by 'meta.cmdt'.
tests=$(find ./tests -name '*.cmdt')

set -x

go build .

transcript check $tests
