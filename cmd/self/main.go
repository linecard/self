package main

import (
	"github.com/linecard/self/cmd/cli"
	"github.com/linecard/self/cmd/handler"
	"github.com/linecard/self/internal/tracing"
	"github.com/linecard/self/internal/util"
)

func main() {
	util.SetLogLevel()

	ctx, tp, shutdown := tracing.InitOtel()
	defer shutdown()

	if util.InLambda() {
		handler.Listen(tp)
		return
	}

	cli.Invoke(ctx)
}
