package config

import (
	"encoding/json"
	"net/url"
	"path/filepath"
	"strings"

	"github.com/linecard/self/internal/gitlib"
	"github.com/linecard/self/internal/util"
	"github.com/linecard/self/pkg/convention/manifest"
)

type ComputedRegistry struct {
	AccountId string
	Region    string
	Url       string
}

type ComputedRepository struct {
	Prefix string
	Name   string
	Path   string
	Url    string
}

type ComputedResource struct {
	Prefix string
	Name   string
	Policy struct {
		Arn string
	}
	Role struct {
		Arn string
	}
}

type ComputedResources struct {
	EphemeralStorage int32  `json:"ephemeralStorage"`
	MemorySize       int32  `json:"memorySize"`
	Timeout          int32  `json:"timeout"`
	Http             bool   `json:"http"`
	Public           bool   `json:"public"`
	RouteKey         string `json:"routeKey"`
}

type Computed struct {
	Registry   ComputedRegistry
	Repository ComputedRepository
	Resource   ComputedResource
	Resources  ComputedResources
}

func (c Config) SolveBuildTime(buildtime manifest.BuildTime) (m manifest.BuildTime, derived Computed, err error) {
	derived.Registry.Solve(c.Account.Id, c.Account.Region)
	derived.Repository.Solve(derived.Registry, c.Git, buildtime.Name.Content)
	derived.Resource.Solve(c.Account.Id, derived.Repository, c.Git, buildtime.Name.Content)
	derived.Resources.Solve(buildtime.Resources.Content, derived.Repository, c.Git, buildtime.Name.Content)
	return buildtime, derived, nil
}

func (c Config) SolveDeployTime(deploytime manifest.DeployTime) (m manifest.DeployTime, derived Computed, err error) {
	origin, err := url.Parse(deploytime.Origin.Content)
	if err != nil {
		return
	}

	git := gitlib.DotGit{
		Branch: deploytime.Branch.Content,
		Origin: origin,
	}

	derived.Registry.Solve(c.Account.Id, c.Account.Region)
	derived.Repository.Solve(derived.Registry, git, deploytime.Name.Content)
	derived.Resource.Solve(c.Account.Id, derived.Repository, git, deploytime.Name.Content)
	derived.Resources.Solve(deploytime.Resources.Content, derived.Repository, git, deploytime.Name.Content)
	return
}

func (registry *ComputedRegistry) Solve(registryId string, registryRegion string) {
	registry.Url = registryId + ".dkr.ecr." + registryRegion + ".amazonaws.com"
}

func (repository *ComputedRepository) Solve(registry ComputedRegistry, git gitlib.DotGit, name string) {
	repository.Prefix = strings.TrimSuffix(git.Origin.Path, ".git")
	repository.Path = filepath.Clean(repository.Prefix + "/" + name)
	repository.Name = repository.Path + "/" + name
	repository.Url = registry.Url + "/" + repository.Path
}

func (resource *ComputedResource) Solve(accountId string, repository ComputedRepository, git gitlib.DotGit, name string) {
	resource.Prefix = util.DeSlasher(repository.Prefix) + "-" + util.DeSlasher(git.Branch)
	resource.Name = resource.Prefix + "-" + name
	resource.Policy.Arn = "arn:aws:iam::" + accountId + ":policy/" + resource.Name
	resource.Role.Arn = "arn:aws:iam::" + accountId + ":role/" + resource.Name
}

func (resources *ComputedResources) Solve(resourcesJson string, repository ComputedRepository, git gitlib.DotGit, name string) {
	defaults := ComputedResources{
		EphemeralStorage: 512,
		MemorySize:       128,
		Timeout:          3,
		Http:             true,
		Public:           false,
		RouteKey:         "ANY /" + repository.Prefix + "/" + git.Branch + "/" + name,
	}

	if resourcesJson != "" {
		if err := json.Unmarshal([]byte(resourcesJson), &resources); err != nil {
			resources = &defaults
		}
	} else {
		resources = &defaults
	}
}
