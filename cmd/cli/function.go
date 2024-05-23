package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/alexflint/go-arg"
	"github.com/jedib0t/go-pretty/table"
	"github.com/linecard/self/internal/labelgun"
)

type UntagOpts struct {
	Tag string `arg:"-t,--tag,env:DEFAULT_RELEASE_BRANCH"`
}

type InspectOpts struct {
	Tag string `arg:"-t,--tag,env:DEFAULT_RELEASE_BRANCH"`
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
	Buses   *NullCommand    `arg:"subcommand:buses" help:"List deployment bus subscriptions"`
	Inspect *InspectOpts    `arg:"subcommand:inspect" help:"Inspect a release"`
}

func (f FunctionScope) Handle(ctx context.Context) {
	if f.Login {
		if err := api.Account.LoginToEcr(ctx); err != nil {
			log.Fatal(err.Error())
		}
	}

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
		log.Fatal(err.Error())
	}

	content, err := labelgun.DecodeLabel(cfg.Label.Policy, image.Config.Labels)
	if err != nil {
		log.Fatal(err.Error())
	}

	policy, err := cfg.Template(content)
	if err != nil {
		log.Fatal(err.Error())
	}

	session, err := cfg.AssumeRoleWithPolicy(ctx, stsc, policy)
	if err != nil {
		log.Fatal(err.Error())
	}

	err = api.Runtime.Emulate(ctx, image, session)
	if err != nil {
		log.Fatal(err.Error())
	}
}

func (f FunctionScope) DeployRelease(ctx context.Context) {
	release, err := api.Release.Find(ctx, f.Deploy.Tag)
	if err != nil {
		log.Fatal(err.Error())
	}

	deployment, err := api.Deployment.Deploy(ctx, release, f.Deploy.NameSpace, cfg.Function.Name)
	if err != nil {
		log.Fatal(err.Error())
	}

	err = api.Subscription.Converge(ctx, deployment)
	if err != nil {
		log.Fatal(err.Error())
	}

	err = api.Httproxy.Converge(ctx, deployment, f.Deploy.NameSpace)
	if err != nil {
		log.Fatal(err.Error())
	}
}

func (f FunctionScope) DestroyDeployment(ctx context.Context) {
	deployment, err := api.Deployment.Find(ctx, f.Destroy.NameSpace, cfg.Function.Name)
	if err != nil {
		log.Fatal(err.Error())
	}

	err = api.Subscription.DisableAll(ctx, deployment)
	if err != nil {
		log.Fatal(err.Error())
	}

	err = api.Httproxy.Unmount(ctx, deployment)
	if err != nil {
		log.Fatal(err.Error())
	}

	err = api.Deployment.Destroy(ctx, deployment)
	if err != nil {
		log.Fatal(err.Error())
	}
}

func (f FunctionScope) EnableDeployment(ctx context.Context) {
	deployment, err := api.Deployment.Find(ctx, f.Enable.NameSpace, cfg.Function.Name)
	if err != nil {
		log.Fatal(err.Error())
	}

	subscriptions, err := api.Subscription.List(ctx, deployment)
	if err != nil {
		log.Fatal(err.Error())
	}

	for _, subscription := range subscriptions {
		err = api.Subscription.Enable(ctx, deployment, subscription)
		if err != nil {
			log.Fatal(err.Error())
		}
	}
}

func (f FunctionScope) DisableDeployment(ctx context.Context) {
	deployment, err := api.Deployment.Find(ctx, f.Disable.NameSpace, cfg.Function.Name)
	if err != nil {
		log.Fatal(err.Error())
	}

	subscriptions, err := api.Subscription.List(ctx, deployment)
	if err != nil {
		log.Fatal(err.Error())
	}

	for _, subscription := range subscriptions {
		err = api.Subscription.Disable(ctx, deployment, subscription)
		if err != nil {
			log.Fatal(err.Error())
		}
	}
}

func (f FunctionScope) PublishRelease(ctx context.Context) {
	if cfg.Git.Dirty {
		log.Fatal("git is dirty, commit changes before publishing")
	}

	path := cfg.Function.Path

	image, err := api.Release.Build(ctx, path, f.Publish.Branch, f.Publish.Sha)
	if err != nil {
		log.Fatal(err.Error())
	}

	err = api.Release.Publish(ctx, image)
	if err != nil {
		log.Fatal(err.Error())
	}
}

func (f FunctionScope) UntagRelease(ctx context.Context) {
	err := api.Release.Untag(ctx, f.Untag.Tag)
	if err != nil {
		log.Fatal(err.Error())
	}
}

func (f FunctionScope) ListBuses(ctx context.Context) {
	deployment, err := api.Deployment.Find(ctx, cfg.Git.Branch, cfg.Function.Name)
	if err != nil {
		log.Fatal(err.Error())
	}

	subscriptions, err := api.Subscription.List(ctx, deployment)
	if err != nil {
		log.Fatal(err.Error())
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
		log.Fatal(err.Error())
	}

	releaseJson, err := json.Marshal(release)
	if err != nil {
		log.Fatal(err.Error())
	}

	fmt.Println(string(releaseJson))
}
