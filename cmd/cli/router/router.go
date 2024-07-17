package router

import (
	"context"
	"os"

	"github.com/linecard/self/cmd/cli/method"
	"github.com/linecard/self/cmd/cli/param"
	"github.com/linecard/self/pkg/convention/config"
	"github.com/linecard/self/pkg/sdk"

	"github.com/alexflint/go-arg"
)

type Root struct {
	Init        *param.Init        `arg:"subcommand:init" help:"Initialize a new function"`
	Build       *param.Build       `arg:"subcommand:build" help:"Build a function"`
	Release     *param.Release     `arg:"subcommand:release" help:"Publish a release"`
	Releases    *param.Releases    `arg:"subcommand:releases" help:"List releases"`
	Deploy      *param.Deploy      `arg:"subcommand:deploy" help:"Deploy a release"`
	Deployments *param.Deployments `arg:"subcommand:deployments" help:"List release deployments"`
	Destroy     *param.Destroy     `arg:"subcommand:destroy" help:"Destroy a release deployment"`
	Config      *param.Config      `arg:"subcommand:config" help:"Print configuration"`
}

func (c Root) Handle(ctx context.Context, cfg config.Config, api sdk.API) {
	switch {
	case c.Init != nil:
		method.InitFunction(ctx, api, c.Init)

	case c.Build != nil:
		method.BuildRelease(ctx, cfg, api, c.Build)

	case c.Release != nil:
		method.PublishRelease(ctx, cfg, api, c.Release)

	case c.Releases != nil:
		method.ListReleases(ctx, cfg, api, c.Releases)

	case c.Deploy != nil:
		method.DeployRelease(ctx, cfg, api, c.Deploy)

	case c.Deployments != nil:
		method.ListDeployments(ctx, cfg, api, c.Deployments)

	case c.Destroy != nil:
		method.DestroyDeployment(ctx, cfg, api, c.Destroy)

	case c.Config != nil:
		method.PrintConfig(ctx, cfg, api, c.Config)

	default:
		arg.MustParse(&c).WriteHelp(os.Stdout)

	}
}
