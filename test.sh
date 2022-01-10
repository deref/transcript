#!/bin/bash

export PATH="$PWD:$PATH"

go build .

transcript check $(find ./tests -name '*.cmdt')
