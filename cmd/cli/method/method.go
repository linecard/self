package method

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/linecard/self/cmd/cli/param"
	"github.com/linecard/self/pkg/convention/config"
	"github.com/linecard/self/pkg/sdk"

	"github.com/charmbracelet/lipgloss/table"
	"github.com/rs/zerolog/log"
)

func InitFunction(ctx context.Context, api sdk.API, p *param.Init) {
	if err := api.Account.Config.Scaffold(p.Language, p.Name); err != nil {
		log.Fatal().Err(err).Msg("failed to scaffold function")
	}
}

func BuildRelease(ctx context.Context, cfg config.Config, api sdk.API, p *param.Build) {
	schema, err := cfg.Find(p.FunctionArg.Path)
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
	schema, err := cfg.Find(p.FunctionArg.Path)
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
	schema, err := cfg.Find(p.FunctionArg.Path)
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
	schema, err := cfg.Find(p.FunctionArg.Path)
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
	t := table.New()

	schema, err := cfg.Find(p.FunctionArg.Path)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to find release schema")
	}

	releases, err := api.Release.List(ctx, schema.Computed.Repository.Name)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to list releases")
	}

	t.Headers("HEAD", "SHA", "DIGEST", "RELEASED")
	for _, release := range releases {
		t.Row(release.Branch, release.GitSha, release.ImageDigest, release.Released)
	}

	fmt.Println(t.Render())
}

func ListDeployments(ctx context.Context, cfg config.Config, api sdk.API, p *param.Deployments) {
	t := table.New()

	schema, err := cfg.Find(p.FunctionArg.Path)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to find release schema")
	}

	deployments, err := api.Deployment.List(ctx, schema.Computed.Resource.Prefix)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to list deployments")
	}

	t.Headers("HEAD", "SHA", "DIGEST", "RELEASED")

	for _, deployment := range deployments {
		t.Row(deployment.Tags["branch"], deployment.Tags["sha"], deployment.Tags["digest"], deployment.Tags["released"])
	}

	fmt.Println(t.Render())
}

func InspectRelease(ctx context.Context, cfg config.Config, api sdk.API, p *param.Inspect) {
	schema, err := cfg.Find(p.FunctionArg.Path)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to find release schema")
	}

	release, err := api.Release.Find(ctx, schema.Computed.Repository.Name, p.Branch)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to find release")
	}

	deploytime, err := schema.Decode(
		cfg.Account.Id,
		cfg.Registry.Id,
		cfg.Registry.Region,
		release.Config.Labels,
		release.AWSArchitecture,
		release.Uri,
	)

	if err != nil {
		log.Fatal().Err(err).Msg("failed to decode deploytime schema")
	}

	dJson, err := json.Marshal(deploytime)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to unmarshal deploytime schema")
	}

	fmt.Println(string(dJson))
}

func PrintConfig(ctx context.Context, cfg config.Config, api sdk.API, p *param.Config) {
	cJson, err := cfg.Json(ctx)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to print configuration")
	}

	fmt.Println(cJson)
}
