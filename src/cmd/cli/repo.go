package cli

import (
	"context"
	"fmt"
	"log"
	"os"
	"slices"
	"strings"
	"sync"

	"github.com/linecard/self/convention/config"
	"github.com/linecard/self/convention/deployment"
	"github.com/linecard/self/convention/release"
	"github.com/linecard/self/internal/util"

	"github.com/alexflint/go-arg"
	"github.com/golang-module/carbon/v2"
	"github.com/jedib0t/go-pretty/table"
)

type GcDeploymentOpts struct {
	NameSpace string `arg:"-n,--namespace,env:DEFAULT_DEPLOYMENT_NAMESPACE"`
}

type RepoScope struct {
	Deployments   *NullCommand      `arg:"subcommand:deployments" help:"List deployments"`
	Releases      *NullCommand      `arg:"subcommand:releases" help:"List releases"`
	GcDeployments *GcDeploymentOpts `arg:"subcommand:gc-deployments" help:"Garbage collect deployments"`
	GcReleases    *NullCommand      `arg:"subcommand:gc-releases" help:"Garbage collect releases"`
	Config        *NullCommand      `arg:"subcommand:config" help:"Print configuration"`
	Login         bool              `arg:"-l,--ecr-login" help:"Login to ECR"`
}

func (r RepoScope) Handle(ctx context.Context) {
	if r.Login {
		if err := api.Account.LoginToEcr(ctx); err != nil {
			log.Fatal(err.Error())
		}
	}

	switch {
	case r.Deployments != nil:
		r.ListDeployments(ctx)

	case r.Releases != nil:
		r.ListReleases(ctx)

	case r.Config != nil:
		r.PrintConfig(ctx)

	case r.GcDeployments != nil:
		r.GcLambda(ctx)

	case r.GcReleases != nil:
		r.GcEcr(ctx)

	default:
		arg.MustParse(&r).WriteUsage(os.Stdout)

	}
}

func (r RepoScope) ListDeployments(ctx context.Context) {
	var wg sync.WaitGroup

	deployments, err := api.Deployment.List(ctx)
	if err != nil {
		log.Fatal(err.Error())
	}

	tablec.AppendHeader(table.Row{"NameSpace", "Function", "Sha", "Digest", "Enabled", "Http", "Updated"})

	wg.Add(len(deployments))

	for _, each := range deployments {
		go func(dep deployment.Deployment) {
			defer wg.Done()
			var enabled bool

			subscriptions, err := api.Subscription.List(ctx, dep)
			if err != nil {
				log.Fatal(err.Error())
			}

			for _, subscription := range subscriptions {
				if subscription.Meta.Update {
					enabled = subscription.Meta.Update
					break
				}
			}

			routes, err := api.Httproxy.ListRoutes(ctx, dep)
			if err != nil {
				log.Fatal(err.Error())
			}

			var routeKeys []string
			for _, route := range routes {
				routeKeys = append(routeKeys, *route.RouteKey)
			}

			tablec.AppendRow(table.Row{
				dep.Tags["NameSpace"],
				dep.Tags["Function"],
				util.SafeSlice(dep.Tags["Sha"], 0, 8),
				util.SafeSlice(*dep.Configuration.CodeSha256, 0, 8),
				enabled,
				strings.Join(routeKeys, ", "),
				carbon.Parse(*dep.Configuration.LastModified).DiffForHumans(),
			})
		}(each)
	}

	wg.Wait()

	tablec.SortBy([]table.SortBy{{Name: "NameSpace", Mode: table.Asc}, {Name: "Function", Mode: table.Asc}})
	tablec.Render()
}

func (r RepoScope) ListReleases(ctx context.Context) {
	var wg sync.WaitGroup

	tablec.AppendHeader(table.Row{"Branch", "Function", "Sha", "Digest", "Released"})

	wg.Add(len(cfg.Functions))
	for _, function := range cfg.Functions {
		go func(f config.Function) {
			defer wg.Done()
			var release release.ReleaseSummary

			releases, err := api.Release.List(ctx, f.Name)
			if err != nil {
				log.Fatal(err.Error())
			}

			for _, release = range releases {
				if release.Branch != "" {
					tablec.AppendRow(table.Row{
						release.Branch,
						f.Name,
						util.SafeSlice(release.GitSha, 0, 8),
						util.SafeSlice(release.ImageDigest, 7, 15),
						carbon.Parse(release.Released).DiffForHumans(),
					})
				}

			}
		}(function)
	}

	wg.Wait()
	tablec.SortBy([]table.SortBy{{Name: "Branch", Mode: table.Asc}})
	tablec.Render()
}

func (r RepoScope) PrintConfig(ctx context.Context) {
	cJson, err := api.Account.Config.Json(ctx)
	if err != nil {
		log.Fatal(err.Error())
	}

	fmt.Println(cJson)
}

func (r RepoScope) GcLambda(ctx context.Context) {
	var delete []deployment.Deployment
	var definedFunctions []string

	deployments, err := api.Deployment.ListNameSpace(ctx, r.GcDeployments.NameSpace)
	if err != nil {
		log.Fatal(err.Error())
	}

	for _, function := range cfg.Functions {
		definedFunctions = append(definedFunctions, function.Name)
	}

	var deploymentMap = make(map[string]deployment.Deployment, len(deployments))

	for _, deployment := range deployments {
		deploymentMap[deployment.Tags["Function"]] = deployment
	}

	for deployedName, deployment := range deploymentMap {
		if !slices.Contains(definedFunctions, deployedName) {
			delete = append(delete, deployment)
		}
	}

	tablec.AppendHeader(table.Row{"NameSpace", "Function", "Sha", "Digest"})

	for _, dep := range delete {
		tablec.AppendRow(table.Row{
			dep.Tags["NameSpace"],
			dep.Tags["Function"],
			util.SafeSlice(dep.Tags["Sha"], 0, 8),
			util.SafeSlice(*dep.Configuration.CodeSha256, 0, 8),
		})
	}

	fmt.Println("destroying following deployments...")
	tablec.SortBy([]table.SortBy{{Name: "Function", Mode: table.Asc}})
	tablec.Render()

	var input string
	fmt.Print("Do you want to run the garbage collection? (y/n): ")
	fmt.Scanf("%s", &input)

	input = strings.TrimSpace(strings.ToLower(input)) // Normalize the input

	if input == "y" {
		for _, d := range delete {
			deployment, err := api.Deployment.Find(ctx, d.Tags["NameSpace"], d.Tags["Function"])
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
		return
	}

	log.Fatal("garbage collection aborted")
}

func (r RepoScope) GcEcr(ctx context.Context) {
	tablec.AppendHeader(table.Row{"Branch", "Function", "Sha", "Digest", "Released"})
	var save []release.ReleaseSummary
	var digestsForDeletion map[string][]string
	var deleteCount int
	var totalCount int
	var err error

	digestsForDeletion = make(map[string][]string, len(cfg.Functions))

	for _, function := range cfg.Functions {
		save, digestsForDeletion[function.Name], err = api.Release.GcPlan(ctx, function.Name)
		if err != nil {
			log.Fatal(err.Error())
		}

		deleteCount += len(digestsForDeletion[function.Name])
		totalCount += len(save) + len(digestsForDeletion[function.Name])

		for _, release := range save {
			tablec.AppendRow(table.Row{
				release.Branch,
				function.Name,
				util.SafeSlice(release.GitSha, 0, 8),
				util.SafeSlice(release.ImageDigest, 7, 15),
				carbon.Parse(release.Released).DiffForHumans(),
			})
		}
	}

	fmt.Printf("deleting %d/%d releases, resulting in...\n", deleteCount, totalCount)
	tablec.SortBy([]table.SortBy{{Name: "Branch", Mode: table.Asc}})
	tablec.Render()

	var input string
	fmt.Print("Do you want to run the garbage collection? (y/n): ")
	fmt.Scanf("%s", &input)

	input = strings.TrimSpace(strings.ToLower(input)) // Normalize the input

	if input == "y" {
		for functionName, digests := range digestsForDeletion {
			if err := api.Release.GcApply(ctx, functionName, digests); err != nil {
				log.Fatal(err.Error())
			}
		}
		return
	}

	log.Fatal("Garbage collection aborted.")
}
