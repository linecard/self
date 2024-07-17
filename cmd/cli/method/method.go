package method

import (
	"context"
	"fmt"

	"github.com/linecard/self/cmd/cli/param"
	"github.com/linecard/self/pkg/convention/config"
	"github.com/linecard/self/pkg/sdk"

	"github.com/rs/zerolog/log"
)

func InitFunction(ctx context.Context, api sdk.API, p *param.Init) {
	if err := api.Account.Config.Scaffold(p.Language, p.Name); err != nil {
		log.Fatal().Err(err).Msg("failed to scaffold function")
	}
}

func BuildRelease(ctx context.Context, cfg config.Config, api sdk.API, p *param.Build) {
	schema, err := cfg.Function(p.FunctionArg.Name)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to find release schema")
	}

	buildtime, err := schema.Encode(cfg)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to encode buildtime schema")
	}

	if _, err := api.Release.Build(ctx, buildtime); err != nil {
		log.Fatal().Err(err).Msg("failed to build release")
	}
}

func PublishRelease(ctx context.Context, cfg config.Config, api sdk.API, p *param.Release) {
	schema, err := cfg.Function(p.FunctionArg.Name)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to find release schema")
	}

	buildtime, err := schema.Encode(cfg)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to encode buildtime schema")
	}

	if p.Login {
		if err := api.Account.LoginToEcr(ctx); err != nil {
			log.Fatal().Err(err).Msg("failed to login to ECR")
		}
	}

	if p.EnsureRepository {
		if err := api.Release.EnsureRepository(ctx, buildtime.Computed.Repository.Name); err != nil {
			log.Fatal().Err(err).Msg("failed to ensure ECR repository")
		}
	}

	image, err := api.Release.Build(ctx, buildtime)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to build release")
	}

	if err := api.Release.Publish(ctx, image); err != nil {
		log.Fatal().Err(err).Msg("failed to publish release")
	}
}

func DeployRelease(ctx context.Context, cfg config.Config, api sdk.API, p *param.Deploy) {
	schema, err := cfg.Function(p.FunctionArg.Name)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to find release schema")
	}

	release, err := api.Release.Find(ctx, schema.Computed.Repository.Name, p.Branch)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to find release")
	}

	deployment, err := api.Deployment.Deploy(ctx, release)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to deploy release")
	}

	if err = api.Subscription.Converge(ctx, deployment); err != nil {
		log.Fatal().Err(err).Msg("failed to converge subscriptions")
	}

	if err = api.Httproxy.Converge(ctx, deployment); err != nil {
		log.Fatal().Err(err).Msg("failed to converge gateway httproxy")
	}
}

func DestroyDeployment(ctx context.Context, cfg config.Config, api sdk.API, p *param.Destroy) {
	schema, err := cfg.Function(p.FunctionArg.Name)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to find release schema")
	}

	deployment, err := api.Deployment.Find(ctx, schema.Computed.Resource.Name)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to find deployment")
	}

	if err = api.Httproxy.Unmount(ctx, deployment); err != nil {
		log.Fatal().Err(err).Msg("failed to unmount gateway httproxy")
	}

	if err = api.Subscription.DisableAll(ctx, deployment); err != nil {
		log.Fatal().Err(err).Msg("failed to disable subscriptions")
	}

	if err = api.Deployment.Destroy(ctx, deployment); err != nil {
		log.Fatal().Err(err).Msg("failed to destroy deployment")
	}
}

func ListReleases(ctx context.Context, cfg config.Config, api sdk.API, p *param.Releases) {
	panic("not implemented")
}

func ListDeployments(ctx context.Context, cfg config.Config, api sdk.API, p *param.Deployments) {
	panic("not implemented")
}

func PrintConfig(ctx context.Context, cfg config.Config, api sdk.API, p *param.Config) {
	cJson, err := cfg.Json(ctx)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to print configuration")
	}

	fmt.Println(cJson)
}
