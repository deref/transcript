package main

import (
	"fmt"
	"os"
)

func main() {
	fmt.Fprintln(os.Stdout, "stdout line 1")
	fmt.Fprintln(os.Stderr, "stderr line 1")
	fmt.Fprintln(os.Stdout, "stdout line 2")
	fmt.Fprintln(os.Stderr, "stderr line 2")
}