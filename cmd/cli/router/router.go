package router

import (
	"context"
	"os"

	"github.com/linecard/self/cmd/cli/method"
	"github.com/linecard/self/cmd/cli/param"
	"github.com/linecard/self/pkg/sdk"

	"github.com/alexflint/go-arg"
)

type Root struct {
	param.GitOpts
	Init        *param.Init        `arg:"subcommand:init" help:"Initialize a new function"`
	Build       *param.Build       `arg:"subcommand:build" help:"Build a function"`
	Publish     *param.Publish     `arg:"subcommand:publish" help:"Publish a release"`
	Releases    *param.Releases    `arg:"subcommand:releases" help:"List releases"`
	Deploy      *param.Deploy      `arg:"subcommand:deploy" help:"Deploy a release"`
	Deployments *param.Deployments `arg:"subcommand:deployments" help:"List release deployments"`
	Destroy     *param.Destroy     `arg:"subcommand:destroy" help:"Destroy a release deployment"`
	Inspect     *param.Inspect     `arg:"subcommand:inspect" help:"Inspect config"`
}

func (c Root) Route(ctx context.Context, api sdk.API) {
	switch {
	case c.Init != nil:
		method.InitFunction(ctx, api, c.Init)

	case c.Build != nil:
		method.BuildRelease(ctx, api, c.Build)

	case c.Publish != nil:
		method.PublishRelease(ctx, api, c.Publish)

	case c.Releases != nil:
		method.ListReleases(ctx, api, c.Releases)

	case c.Deploy != nil:
		method.DeployRelease(ctx, api, c.Deploy)

	case c.Deployments != nil:
		method.ListDeployments(ctx, api, c.Deployments)

	case c.Destroy != nil:
		method.DestroyDeployment(ctx, api, c.Destroy)

	case c.Inspect != nil:
		switch {
		case c.Inspect.Build != nil:
			method.PrintBuildTime(ctx, api, c.Inspect.Build)

		case c.Inspect.Deploy != nil:
			method.PrintDeployTime(ctx, api, c.Inspect.Deploy)

		default:
			arg.MustParse(&c).WriteHelpForSubcommand(os.Stdout, "inspect")
		}

	default:
		arg.MustParse(&c).WriteHelp(os.Stdout)

	}
}
