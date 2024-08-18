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
	"github.com/linecard/self/pkg/convention/config"
	dtype "github.com/linecard/self/pkg/convention/deployment"
	"github.com/linecard/self/pkg/sdk"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/propagation"

	"github.com/charmbracelet/lipgloss/table"
	"github.com/rs/zerolog/log"
)

func InitFunction(ctx context.Context, api sdk.API, p *param.Init) error {
	if err := api.Account.Config.Scaffold(p.Scaffold, p.Name); err != nil {
		return err
	}
	return nil
}

func BuildRelease(ctx context.Context, api sdk.API, p *param.Build) error {
	if p.Context == "" {
		p.Context = p.Path
	}

	image, _, err := api.Release.Build(ctx, p.Path, p.Context)
	if err != nil {
		return err
	}

	if p.Run {
		if err := api.Runtime.Emulate(ctx, image); err != nil {
			return err
		}
	}

	return nil
}

func PublishRelease(ctx context.Context, api sdk.API, p *param.Publish) error {
	ctx, span := otel.Tracer("").Start(ctx, "release")
	defer span.End()

	if p.Context == "" {
		p.Context = p.Path
	}

	if p.Login {
		if err := api.Account.LoginToEcr(ctx); err != nil {
			return err
		}
	}

	image, buildtime, err := api.Release.Build(ctx, p.Path, p.Context)
	if err != nil {
		return err
	}

	if p.EnsureRepository {
		if err := api.Release.EnsureRepository(ctx, buildtime.Computed.Repository.Name); err != nil {
			return err
		}
	}

	if api.Config.Git.Dirty && !p.Force {
		log.Fatal().Msg("git is dirty, please commit changes before publishing")
	}

	if err := api.Release.Publish(ctx, image); err != nil {
		return err
	}

	if p.EmitDeploy {
		ctx, span := otel.Tracer("").Start(ctx, "notify")
		defer span.End()

		carrier := propagation.MapCarrier{}
		otel.GetTextMapPropagator().Inject(ctx, carrier)

		detail := config.EventDetail{
			Traceparent:    carrier["traceparent"],
			Tracestate:     carrier["tracestate"],
			Action:         "Deploy",
			Sha:            buildtime.Sha.Decoded,
			Branch:         buildtime.Branch.Decoded,
			Origin:         buildtime.Origin.Decoded,
			RepositoryName: buildtime.Computed.Repository.Name,
			ResourceName:   buildtime.Computed.Resource.Name,
		}

		if err := api.Bus.Emit(ctx, detail); err != nil {
			return err
		}
	}

	return nil
}

func DeployRelease(ctx context.Context, api sdk.API, p *param.Deploy) error {
	if p.Enable && p.Disable {
		log.Fatal().Msg("--enable and --disable are mutually exclusive")
	}

	buildtime, err := api.Config.BuildTime(p.FunctionArg.Path)
	if err != nil {
		return err
	}

	release, err := api.Release.Find(ctx, buildtime.Computed.Repository.Name, api.Config.Git.Branch)
	if err != nil {
		return err
	}

	deployment, err := api.Deployment.Deploy(ctx, release)
	if err != nil {
		return err
	}

	if p.Enable {
		if err = api.Subscription.EnableAll(ctx, deployment); err != nil {
			return err
		}
	}

	if p.Disable {
		if err = api.Subscription.DisableAll(ctx, deployment); err != nil {
			return err
		}
	}

	if err = api.Subscription.Converge(ctx, deployment); err != nil {
		return err
	}

	if err = api.Httproxy.Converge(ctx, deployment); err != nil {
		return err
	}

	return nil
}

func DestroyDeployment(ctx context.Context, api sdk.API, p *param.Destroy) error {
	buildtime, err := api.Config.BuildTime(p.FunctionArg.Path)
	if err != nil {
		return err
	}

	deployment, err := api.Deployment.Find(ctx, buildtime.Computed.Resource.Name)
	if err != nil {
		return err
	}

	if err = api.Httproxy.Unmount(ctx, deployment); err != nil {
		return err
	}

	if err = api.Subscription.DisableAll(ctx, deployment); err != nil {
		return err
	}

	if err = api.Deployment.Destroy(ctx, deployment); err != nil {
		return err
	}

	return nil
}

func UntagRelease(ctx context.Context, api sdk.API, p *param.Untag) error {
	ctx, span := otel.Tracer("").Start(ctx, "release")
	defer span.End()

	buildtime, err := api.Config.BuildTime(p.FunctionArg.Path)
	if err != nil {
		return err
	}

	span.SetAttributes(
		attribute.String("branch", buildtime.Branch.Decoded),
		attribute.String("sha", buildtime.Sha.Decoded),
		attribute.String("origin", buildtime.Origin.Decoded),
		attribute.String("repository", buildtime.Computed.Repository.Name),
		attribute.String("resource", buildtime.Computed.Resource.Name),
	)

	err = api.Release.Untag(ctx, buildtime.Computed.Repository.Name, api.Config.Git.Branch)
	if err != nil {
		return err
	}

	if p.EmitDestroy {
		ctx, span := otel.Tracer("").Start(ctx, "notify")
		defer span.End()

		carrier := propagation.MapCarrier{}
		otel.GetTextMapPropagator().Inject(ctx, carrier)

		detail := config.EventDetail{
			Traceparent:    carrier["traceparent"],
			Tracestate:     carrier["tracestate"],
			Action:         "Destroy",
			Sha:            buildtime.Sha.Decoded,
			Branch:         buildtime.Branch.Decoded,
			Origin:         buildtime.Origin.Decoded,
			RepositoryName: buildtime.Computed.Repository.Name,
			ResourceName:   buildtime.Computed.Resource.Name,
		}

		err := api.Bus.Emit(ctx, detail)

		if err != nil {
			return err
		}
	}

	return nil
}

func ListReleases(ctx context.Context, api sdk.API, p *param.Releases) error {
	t := table.New()

	buildtime, err := api.Config.BuildTime(p.FunctionArg.Path)
	if err != nil {
		return err
	}

	releases, err := api.Release.List(ctx, buildtime.Computed.Repository.Name)
	if err != nil {
		return err
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
	return nil
}

func ListDeployments(ctx context.Context, api sdk.API, p *param.Deployments) error {
	var wg sync.WaitGroup
	t := table.New()

	branchFilter := api.Config.Resource.Namespace + "-" + api.Config.Git.Branch
	deployments, err := api.Deployment.List(ctx, branchFilter)
	if err != nil {
		return err
	}

	t.Headers("Deployment", "HEAD", "SHA", "DIGEST", "ENABLED", "ROUTE", "DEPLOYED")

	wg.Add(len(deployments))

	for _, deployment := range deployments {
		go func(each dtype.Deployment) error {
			defer wg.Done()
			var enabled bool
			var err error

			subscriptions, err := api.Subscription.List(ctx, each)
			if err != nil {
				log.Warn().Err(err).Msg("error while to listing subscriptions")
			}

			for _, subscription := range subscriptions {
				if subscription.Meta.Update {
					enabled = subscription.Meta.Update
					break
				}
			}

			routes, err := api.Httproxy.UnsafeListRoutes(ctx, each)
			if err != nil {
				log.Warn().Err(err).Msg("error while to listing routes")
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

			return nil
		}(deployment)
	}

	wg.Wait()

	fmt.Println(t.Render())
	return nil
}

func PrintGlobalConfig(ctx context.Context, api sdk.API) error {
	cJson, err := json.Marshal(api.Config)
	if err != nil {
		return err
	}

	fmt.Println(string(cJson))
	return nil
}

func PrintDeployTime(ctx context.Context, api sdk.API, p *param.DeployTime) error {
	buildtime, err := api.Config.BuildTime(p.FunctionArg.Path)
	if err != nil {
		return err
	}

	release, err := api.Release.Find(ctx, buildtime.Computed.Repository.Name, api.Config.Git.Branch)
	if err != nil {
		return err
	}

	deploytime, err := api.Config.DeployTime(release.Config.Labels)
	if err != nil {
		return err
	}

	out, err := json.Marshal(deploytime)
	if err != nil {
		return err
	}

	fmt.Println(string(out))
	return nil
}

func PrintBuildTime(ctx context.Context, api sdk.API, p *param.BuildTime) error {
	buildtime, err := api.Config.BuildTime(p.FunctionArg.Path)
	if err != nil {
		return err
	}

	out, err := json.Marshal(buildtime)
	if err != nil {
		return err
	}

	fmt.Println(string(out))
	return nil
}
