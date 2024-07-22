package cli

import (
	"context"

	"github.com/linecard/self/cmd/cli/router"
	"github.com/linecard/self/pkg/convention/config"
	"github.com/linecard/self/pkg/sdk"

	"github.com/alexflint/go-arg"
	"github.com/aws/aws-sdk-go-v2/service/sts"
)

var cfg config.Config
var api sdk.API
var stsc *sts.Client

func Invoke(ctx context.Context) {
	BeforeAll(ctx)

	var c router.Root
	arg.MustParse(&c)
	c.Handle(ctx, cfg, api)
}
