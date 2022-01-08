package main

import (
	"context"

	"github.com/deref/transcript/internal/cli"
)

func main() {
	ctx := context.Background()
	cli.Execute(ctx)
}
