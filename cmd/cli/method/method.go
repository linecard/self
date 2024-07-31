package method

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"sync"

	"github.com/golang-module/carbon/v2"
	"github.com/linecard/self/cmd/cli/param"
	"github.com/linecard/self/internal/util"
	dtype "github.com/linecard/self/pkg/convention/deployment"
	"github.com/linecard/self/pkg/sdk"

	"github.com/charmbracelet/lipgloss/table"
	"github.com/rs/zerolog/log"
)

func InitFunction(ctx context.Context, api sdk.API, p *param.Init) {
	if err := api.Account.Config.Scaffold(p.Language, p.Name); err != nil {
		log.Fatal().Err(err).Msg("failed to scaffold function")
	}
}

func BuildRelease(ctx context.Context, api sdk.API, p *param.Build) {
	if p.Context == "" {
		p.Context = p.Path
	}

	image, _, err := api.Release.Build(ctx, p.Path, p.Context)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to build release")
	}

	if p.Run {
		deploytime, err := api.Config.Parse(image.Config.Labels)
		if err != nil {
			log.Fatal().Err(err).Msg("failed to decode deploytime schema")
		}

		creds, err := api.Config.AssumeRoleWithPolicy(ctx, deploytime.Policy.Decoded)
		if err != nil {
			log.Fatal().Err(err).Msg("failed to assume role with policy")
		}

		if err := api.Runtime.Emulate(ctx, image, creds); err != nil {
			log.Fatal().Err(err).Msg("failed to emulate runtime")
		}
	}
}

func PublishRelease(ctx context.Context, api sdk.API, p *param.Publish) {
	if p.Context == "" {
		p.Context = p.Path
	}

	if p.Login {
		if err := api.Account.LoginToEcr(ctx); err != nil {
			log.Fatal().Err(err).Msg("failed to login to ECR")
		}
	}

	image, buildtime, err := api.Release.Build(ctx, p.Path, p.Context)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to build release")
	}

	if p.EnsureRepository {
		if err := api.Release.EnsureRepository(ctx, buildtime.Computed.Repository.Name); err != nil {
			log.Fatal().Err(err).Msg("failed to ensure ECR repository")
		}
	}

	if err := api.Release.Publish(ctx, image); err != nil {
		log.Fatal().Err(err).Msg("failed to publish release")
	}
}

func DeployRelease(ctx context.Context, api sdk.API, p *param.Deploy) {
	if p.Enable && p.Disable {
		log.Fatal().Msg("enable and disable are mutually exclusive")
	}

	buildtime, err := api.Config.Find(p.FunctionArg.Path)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to find release schema")
	}

	release, err := api.Release.Find(ctx, buildtime.Computed.Repository.Name, api.Config.Git.Branch)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to find release")
	}

	deployment, err := api.Deployment.Deploy(ctx, release)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to deploy release")
	}

	if p.Enable {
		if err = api.Subscription.EnableAll(ctx, deployment); err != nil {
			log.Fatal().Err(err).Msg("failed to enable subscriptions")
		}
	}

	if p.Disable {
		if err = api.Subscription.DisableAll(ctx, deployment); err != nil {
			log.Fatal().Err(err).Msg("failed to disable subscriptions")
		}
	}

	if err = api.Subscription.Converge(ctx, deployment); err != nil {
		log.Fatal().Err(err).Msg("failed to converge subscriptions")
	}

	if err = api.Httproxy.Converge(ctx, deployment); err != nil {
		log.Fatal().Err(err).Msg("failed to converge gateway httproxy")
	}
}

func DestroyDeployment(ctx context.Context, api sdk.API, p *param.Destroy) {
	buildtime, err := api.Config.Find(p.FunctionArg.Path)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to find release schema")
	}

	deployment, err := api.Deployment.Find(ctx, buildtime.Computed.Resource.Name)
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

func ListReleases(ctx context.Context, api sdk.API, p *param.Releases) {
	t := table.New()

	buildtime, err := api.Config.Find(p.FunctionArg.Path)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to find release schema")
	}

	releases, err := api.Release.List(ctx, buildtime.Computed.Repository.Name)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to list releases")
	}

	t.Headers("HEAD", "SHA", "DIGEST", "RELEASED")
	for _, release := range releases {
		t.Row(
			release.Branch,
			util.UnsafeSlice(release.GitSha, 0, 8),
			util.UnsafeSlice(release.ImageDigest, 7, 15),
			carbon.Parse(release.Released).DiffForHumans(),
		)
	}

	fmt.Println(t.Render())
}

func ListDeployments(ctx context.Context, api sdk.API, p *param.Deployments) {
	var wg sync.WaitGroup
	t := table.New()

	branchFilter := api.Config.Resource.Namespace + "-" + api.Config.Git.Branch
	deployments, err := api.Deployment.List(ctx, branchFilter)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to list deployments")
	}

	t.Headers("Deployment", "HEAD", "SHA", "DIGEST", "ENABLED", "ROUTE", "DEPLOYED")

	wg.Add(len(deployments))

	for _, deployment := range deployments {
		go func(each dtype.Deployment) {
			defer wg.Done()
			var enabled bool

			subscriptions, err := api.Subscription.List(ctx, each)
			if err != nil {
				log.Fatal().Err(err).Msg("failed to list subscriptions")
			}

			for _, subscription := range subscriptions {
				if subscription.Meta.Update {
					enabled = subscription.Meta.Update
					break
				}
			}

			routes, err := api.Httproxy.UnsafeListRoutes(ctx, each)
			if err != nil {
				log.Fatal().Err(err).Msg("failed to list routes")
			}

			var routeKeys []string
			for _, route := range routes {
				routeKeys = append(routeKeys, *route.RouteKey)
			}

			t.Row(
				each.Tags["Function"],
				each.Tags["Branch"],
				util.UnsafeSlice(each.Tags["Sha"], 0, 8),
				util.UnsafeSlice(*each.Configuration.CodeSha256, 0, 8),
				strconv.FormatBool(enabled),
				strings.Join(routeKeys, ", "),
				carbon.Parse(*each.Configuration.LastModified).DiffForHumans(),
			)
		}(deployment)
	}

	wg.Wait()

	fmt.Println(t.Render())
}

func PrintGlobalConfig(ctx context.Context, api sdk.API) {
	cJson, err := json.Marshal(api.Config)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to marshal config")
	}

	fmt.Println(string(cJson))
}

func PrintDeployTime(ctx context.Context, api sdk.API, p *param.DeployTime) {
	buildtime, err := api.Config.Find(p.FunctionArg.Path)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to find release schema")
	}

	release, err := api.Release.Find(ctx, buildtime.Computed.Repository.Name, api.Config.Git.Branch)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to find release")
	}

	deploytime, err := api.Config.Parse(release.Config.Labels)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to decode deploytime schema")
	}

	out, err := json.Marshal(deploytime)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to marshal deploytime schema")
	}

	fmt.Println(string(out))
}

func PrintBuildTime(ctx context.Context, api sdk.API, p *param.BuildTime) {
	buildtime, err := api.Config.Find(p.FunctionArg.Path)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to find buildtime schema")
	}

	out, err := json.Marshal(buildtime)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to marshal buildtime schema")
	}

	fmt.Println(string(out))
}
