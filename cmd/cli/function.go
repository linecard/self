package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/alexflint/go-arg"
	"github.com/jedib0t/go-pretty/table"

	"github.com/rs/zerolog/log"
)

type UntagOpts struct {
	Tag string `arg:"-t,--tag,env:DEFAULT_RELEASE_BRANCH"`
}

type InspectOpts struct {
	Tag string `arg:"-t,--tag,env:DEFAULT_RELEASE_BRANCH"`
}

type BusesOpts struct {
	NameSpace string `arg:"-n,--namespace,env:DEFAULT_DEPLOYMENT_NAMESPACE"`
}

type FunctionScope struct {
	RepoScope
	Run     *NullCommand    `arg:"subcommand:run" help:"Deploy the function locally"`
	Deploy  *DeploymentOpts `arg:"subcommand:deploy" help:"Deploy function from release"`
	Destroy *DeploymentOpts `arg:"subcommand:destroy" help:"Destroy deployment of release"`
	Enable  *DeploymentOpts `arg:"subcommand:enable" help:"Subscribe deployment to release bus definitions"`
	Disable *DeploymentOpts `arg:"subcommand:disable" help:"Unsubscribe deployment from buses"`
	Publish *ReleaseOpts    `arg:"subcommand:publish" help:"Publish a release"`
	Untag   *UntagOpts      `arg:"subcommand:untag" help:"Untag a release"`
	Buses   *BusesOpts      `arg:"subcommand:buses" help:"List deployment bus subscriptions"`
	Inspect *InspectOpts    `arg:"subcommand:inspect" help:"Inspect a release"`
}

func (f FunctionScope) Handle(ctx context.Context) {
	switch {
	case f.Deployments != nil:
		f.ListDeployments(ctx)

	case f.Releases != nil:
		f.ListReleases(ctx)

	case f.Config != nil:
		f.PrintConfig(ctx)

	case f.Run != nil:
		f.DeployLocal(ctx)

	case f.Deploy != nil:
		f.DeployRelease(ctx)

	case f.Destroy != nil:
		f.DestroyDeployment(ctx)

	case f.Enable != nil:
		f.EnableDeployment(ctx)

	case f.Disable != nil:
		f.DisableDeployment(ctx)

	case f.Publish != nil:
		f.PublishRelease(ctx)

	case f.Untag != nil:
		f.UntagRelease(ctx)

	case f.GcReleases != nil:
		f.GcEcr(ctx)

	case f.GcDeployments != nil:
		f.GcLambda(ctx)

	case f.Buses != nil:
		f.ListBuses(ctx)

	case f.Inspect != nil:
		f.InspectRelease(ctx)

	default:
		arg.MustParse(&f).WriteUsage(os.Stdout)

	}
}

func (f FunctionScope) DeployLocal(ctx context.Context) {
	image, err := api.Release.Build(ctx, cfg.Function.Path, cfg.Git.Branch, cfg.Git.Sha)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to build image")
	}

	labels, err := cfg.Labels.Decode(image.Config.Labels)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to decode labels")
	}

	for k, v := range labels {
		templatedValue, err := cfg.Template(v)
		if err != nil {
			log.Fatal().Err(err).Msg("failed to render label template")
		}

		labels[k] = templatedValue
	}

	policy, err := cfg.Template(labels[cfg.Labels.Policy.Key])
	if err != nil {
		log.Fatal().Err(err).Msg("failed to render policy template")
	}

	session, err := cfg.AssumeRoleWithPolicy(ctx, stsc, policy)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to assume role with policy, is your IAM role able to assume itself?")
	}

	err = api.Runtime.Emulate(ctx, image, session)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to emulate function")
	}
}

func (f FunctionScope) DeployRelease(ctx context.Context) {
	release, err := api.Release.Find(ctx, f.Deploy.Tag)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to find release")
	}

	deployment, err := api.Deployment.Deploy(ctx, release, f.Deploy.NameSpace, cfg.Function.Name)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to deploy release")
	}

	err = api.Subscription.Converge(ctx, deployment)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to converge subscriptions")
	}

	err = api.Httproxy.Converge(ctx, deployment, f.Deploy.NameSpace)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to converge gateway httproxy")
	}
}

func (f FunctionScope) DestroyDeployment(ctx context.Context) {
	deployment, err := api.Deployment.Find(ctx, f.Destroy.NameSpace, cfg.Function.Name)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to find deployment")
	}

	err = api.Subscription.DisableAll(ctx, deployment)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to disable subscriptions")
	}

	err = api.Httproxy.Unmount(ctx, deployment)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to unmount gateway httproxy")
	}

	err = api.Deployment.Destroy(ctx, deployment)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to destroy deployment")
	}
}

func (f FunctionScope) EnableDeployment(ctx context.Context) {
	deployment, err := api.Deployment.Find(ctx, f.Enable.NameSpace, cfg.Function.Name)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to find deployment")
	}

	subscriptions, err := api.Subscription.List(ctx, deployment)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to list subscriptions")
	}

	for _, subscription := range subscriptions {
		err = api.Subscription.Enable(ctx, deployment, subscription)
		if err != nil {
			log.Fatal().Err(err).Msg("failed to enable subscription")
		}
	}
}

func (f FunctionScope) DisableDeployment(ctx context.Context) {
	deployment, err := api.Deployment.Find(ctx, f.Disable.NameSpace, cfg.Function.Name)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to find deployment")
	}

	subscriptions, err := api.Subscription.List(ctx, deployment)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to list subscriptions")
	}

	for _, subscription := range subscriptions {
		err = api.Subscription.Disable(ctx, deployment, subscription)
		if err != nil {
			log.Fatal().Err(err).Msg("failed to disable subscription")
		}
	}
}

func (f FunctionScope) PublishRelease(ctx context.Context) {
	if cfg.Git.Dirty {
		log.Fatal().Msg("refusing to publish dirty branch state")
	}

	if f.Publish.Login {
		if err := api.Account.LoginToEcr(ctx); err != nil {
			log.Fatal().Err(err).Msg("failed to login to ecr")
		}
	}

	if f.Publish.EnsureRepository {
		if err := api.Release.EnsureRepository(ctx); err != nil {
			log.Fatal().Err(err).Msg("failed to ensure repository")
		}
	}

	path := cfg.Function.Path

	image, err := api.Release.Build(ctx, path, f.Publish.Branch, f.Publish.Sha)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to build image")
	}

	err = api.Release.Publish(ctx, image)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to publish release")
	}
}

func (f FunctionScope) UntagRelease(ctx context.Context) {
	err := api.Release.Untag(ctx, f.Untag.Tag)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to untag release")
	}
}

func (f FunctionScope) ListBuses(ctx context.Context) {
	deployment, err := api.Deployment.Find(ctx, f.Buses.NameSpace, cfg.Function.Name)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to find deployment")
	}

	subscriptions, err := api.Subscription.List(ctx, deployment)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to list subscriptions")
	}

	tablec.AppendHeader(table.Row{"Bus", "Rule", "Destroy", "Update", "Reason"})

	for _, subscription := range subscriptions {
		tablec.AppendRow(table.Row{
			subscription.Meta.Bus,
			subscription.Meta.Rule,
			subscription.Meta.Destroy,
			subscription.Meta.Update,
			subscription.Meta.Reason,
		})
	}

	tablec.SortBy([]table.SortBy{{Name: "Bus", Mode: table.Asc}})
	tablec.Render()
}

func (f FunctionScope) InspectRelease(ctx context.Context) {
	release, err := api.Release.Find(ctx, f.Inspect.Tag)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to find release")
	}

	releaseJson, err := json.Marshal(release)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to marshal release struct to json")
	}

	fmt.Println(string(releaseJson))
}
