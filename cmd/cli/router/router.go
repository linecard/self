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
	param.GlobalOpts
	Init        *param.Init        `arg:"subcommand:init" help:"Initialize a scaffold"`
	Build       *param.Build       `arg:"subcommand:build" help:"Build a release"`
	Publish     *param.Publish     `arg:"subcommand:publish" help:"Publish a release"`
	Deploy      *param.Deploy      `arg:"subcommand:deploy" help:"Deploy a release"`
	Destroy     *param.Destroy     `arg:"subcommand:destroy" help:"Destroy a release deployment"`
	Releases    *param.Releases    `arg:"subcommand:releases" help:"List releases"`
	Deployments *param.Deployments `arg:"subcommand:deployments" help:"List release deployments"`
	Inspect     *param.Inspect     `arg:"subcommand:inspect" help:"Inspect config"`
	Untag       *param.Untag       `arg:"subcommand:untag" help:"Untag a release"`
}

func (c Root) Route(ctx context.Context, api sdk.API) error {
	switch {
	case c.Init != nil:
		return method.InitFunction(ctx, api, c.Init)

	case c.Build != nil:
		return method.BuildRelease(ctx, api, c.Build)

	case c.Publish != nil:
		return method.PublishRelease(ctx, api, c.Publish)

	case c.Releases != nil:
		return method.ListReleases(ctx, api, c.Releases)

	case c.Deploy != nil:
		return method.DeployRelease(ctx, api, c.Deploy)

	case c.Deployments != nil:
		return method.ListDeployments(ctx, api, c.Deployments)

	case c.Destroy != nil:
		return method.DestroyDeployment(ctx, api, c.Destroy)

	case c.Untag != nil:
		return method.UntagRelease(ctx, api, c.Untag)

	case c.Inspect != nil:
		switch {
		case c.Inspect.Build != nil:
			return method.PrintBuildTime(ctx, api, c.Inspect.Build)

		case c.Inspect.Deploy != nil:
			return method.PrintDeployTime(ctx, api, c.Inspect.Deploy)

		case c.Inspect.Global != nil:
			return method.PrintGlobalConfig(ctx, api)

		default:
			arg.MustParse(&c).WriteHelpForSubcommand(os.Stdout, "inspect")
			return nil
		}

	default:
		arg.MustParse(&c).WriteHelp(os.Stdout)
		return nil

	}
}
