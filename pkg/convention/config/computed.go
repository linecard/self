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
	Url string
}

type ComputedRepository struct {
	Namespace string
	Name      string
	Url       string
}

type ComputedResource struct {
	Namespace string
	Name      string
	Policy    struct {
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

func (c Computed) Json() (string, error) {
	b, err := json.Marshal(c)
	return string(b), err
}

func (c Config) SolveBuildTime(buildtime manifest.BuildTime) (m manifest.BuildTime, derived Computed, err error) {
	derived.Registry.Solve(c.Account.Id, c.Account.Region)
	derived.Repository.Solve(derived.Registry, c.Git, buildtime.Name.Decoded)
	derived.Resource.Solve(c.Account.Id, derived.Repository, c.Git, buildtime.Name.Decoded)
	derived.Resources.Solve(buildtime.Resources.Decoded, derived.Repository, c.Git, buildtime.Name.Decoded)
	return buildtime, derived, nil
}

func (c Config) SolveDeployTime(deploytime manifest.DeployTime) (m manifest.DeployTime, derived Computed, err error) {
	origin, err := url.Parse(deploytime.Origin.Decoded)
	if err != nil {
		return
	}

	git := gitlib.DotGit{
		Branch: deploytime.Branch.Decoded,
		Origin: origin,
	}

	derived.Registry.Solve(c.Account.Id, c.Account.Region)
	derived.Repository.Solve(derived.Registry, git, deploytime.Name.Decoded)
	derived.Resource.Solve(c.Account.Id, derived.Repository, git, deploytime.Name.Decoded)
	derived.Resources.Solve(deploytime.Resources.Decoded, derived.Repository, git, deploytime.Name.Decoded)
	return deploytime, derived, nil
}

func (registry *ComputedRegistry) Solve(registryId string, registryRegion string) {
	registry.Url = registryId + ".dkr.ecr." + registryRegion + ".amazonaws.com"
}

func (repository *ComputedRepository) Solve(registry ComputedRegistry, git gitlib.DotGit, name string) {
	nameSpace := strings.TrimSuffix(git.Origin.Path, ".git")
	repository.Namespace = strings.TrimPrefix(nameSpace, "/")
	repository.Name = filepath.Clean(repository.Namespace + "/" + name)
	repository.Url = registry.Url + "/" + repository.Name
}

func (resource *ComputedResource) Solve(accountId string, repository ComputedRepository, git gitlib.DotGit, name string) {
	resource.Namespace = util.DeSlasher(repository.Namespace) + "-" + git.Branch
	resource.Name = resource.Namespace + "-" + name
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
		RouteKey:         "ANY /" + repository.Namespace + "/" + git.Branch + "/" + name,
	}

	// Start with the default values
	*resources = defaults

	if resourcesJson != "" {
		// Unmarshal into a temporary struct
		var temp ComputedResources
		if err := json.Unmarshal([]byte(resourcesJson), &temp); err == nil {
			// If unmarshaling succeeds, update only the non-zero values
			if temp.EphemeralStorage != 0 {
				resources.EphemeralStorage = temp.EphemeralStorage
			}
			if temp.MemorySize != 0 {
				resources.MemorySize = temp.MemorySize
			}
			if temp.Timeout != 0 {
				resources.Timeout = temp.Timeout
			}
			// For boolean fields, we need to check if they were explicitly set in the JSON
			if resourcesJson != "" {
				var jsonMap map[string]interface{}
				json.Unmarshal([]byte(resourcesJson), &jsonMap)
				if _, ok := jsonMap["http"]; ok {
					resources.Http = temp.Http
				}
				if _, ok := jsonMap["public"]; ok {
					resources.Public = temp.Public
				}
			}
			if temp.RouteKey != "" {
				resources.RouteKey = temp.RouteKey
			}
		}
	}
}
