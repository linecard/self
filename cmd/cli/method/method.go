package method

import (
	"context"
	"fmt"

	"github.com/linecard/self/cmd/cli/param"
	"github.com/linecard/self/cmd/cli/view"
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
	buildtime, computed, err := cfg.Find(p.FunctionArg.Path)
	if err != nil {
		log.Fatal().Err(err).Msgf("failed lookup for %s", p.FunctionArg.Path)
	}

	if _, err := api.Release.Build(ctx, buildtime, computed); err != nil {
		log.Fatal().Err(err).Msg("failed to build release")
	}
}

func PublishRelease(ctx context.Context, cfg config.Config, api sdk.API, p *param.Publish) {
	buildtime, computed, err := cfg.Find(p.FunctionArg.Path)
	if err != nil {
		log.Fatal().Err(err).Msgf("failed lookup for %s", p.FunctionArg.Path)
	}

	if p.Login {
		if err := api.Account.LoginToEcr(ctx); err != nil {
			log.Fatal().Err(err).Msg("failed to login to ECR")
		}
	}

	if p.EnsureRepository {
		if err := api.Release.EnsureRepository(ctx, computed.Repository.Name); err != nil {
			log.Fatal().Err(err).Msg("failed to ensure ECR repository")
		}
	}

	image, err := api.Release.Build(ctx, buildtime, computed)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to build release")
	}

	if err := api.Release.Publish(ctx, image); err != nil {
		log.Fatal().Err(err).Msg("failed to publish release")
	}
}

func DeployRelease(ctx context.Context, cfg config.Config, api sdk.API, p *param.Deploy) {
	_, computed, err := cfg.Find(p.FunctionArg.Path)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to find release schema")
	}

	release, err := api.Release.Find(ctx, computed.Repository.Name, p.Branch)
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
	_, computed, err := cfg.Find(p.FunctionArg.Path)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to find release schema")
	}

	deployment, err := api.Deployment.Find(ctx, computed.Resource.Name)
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

	_, computed, err := cfg.Find(p.FunctionArg.Path)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to find release schema")
	}

	releases, err := api.Release.List(ctx, computed.Repository.Name)
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

	_, computed, err := cfg.Find(p.FunctionArg.Path)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to find release schema")
	}

	deployments, err := api.Deployment.List(ctx, computed.Resource.Namespace)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to list deployments")
	}

	t.Headers("HEAD", "SHA", "DIGEST", "RELEASED")

	for _, deployment := range deployments {
		t.Row(deployment.Tags["branch"], deployment.Tags["sha"], deployment.Tags["digest"], deployment.Tags["released"])
	}

	fmt.Println(t.Render())
}

func PrintDeployTime(ctx context.Context, cfg config.Config, api sdk.API, p *param.DeployTime) {
	_, computed, err := cfg.Find(p.FunctionArg.Path)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to find release schema")
	}

	release, err := api.Release.Find(ctx, computed.Repository.Name, p.Branch)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to find release")
	}

	deploytime, computed, err := cfg.Parse(release.Config.Labels)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to decode deploytime schema")
	}

	v := view.DeployTimeView{
		Manifest: deploytime,
		Computed: computed,
	}

	vJson, err := v.Json()
	if err != nil {
		log.Fatal().Err(err).Msg("failed to marshal deploytime view data")
	}

	fmt.Println(string(vJson))
}

func PrintBuildTime(ctx context.Context, cfg config.Config, api sdk.API, p *param.BuildTime) {
	if p.Global {
		computedJson, err := cfg.Json(ctx)
		if err != nil {
			log.Fatal().Err(err).Msg("failed to print configuration")
		}

		fmt.Println(computedJson)
		return
	}

	buildtime, computed, err := cfg.Find(p.FunctionArg.Path)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to find buildtime schema")
	}

	v := view.BuildTimeView{
		Manifest: buildtime,
		Computed: computed,
	}

	viewJson, err := v.Json()
	if err != nil {
		log.Fatal().Err(err).Msg("failed to marshal buildtime view data")
	}

	fmt.Println(viewJson)
}
