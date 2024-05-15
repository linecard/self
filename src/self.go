package main

import (
	"context"

	"github.com/linecard/self/cmd/cli"
	"github.com/linecard/self/cmd/handler"
	"github.com/linecard/self/internal/util"
)

func main() {
	ctx := context.Background()

	if util.InLambda() {
		handler.Listen(ctx)
		return
	}

	cli.Invoke(ctx)
}
