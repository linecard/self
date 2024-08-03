package config

import (
	"encoding/json"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/linecard/self/internal/gitlib"
	"github.com/linecard/self/pkg/convention/manifest"
)

type BuildTime struct {
	manifest.BuildTime
	Computed Computed
}

type DeployTime struct {
	manifest.DeployTime
	Computed Computed
}

type ComputedRegistry struct {
	Url string
}

type ComputedRepository struct {
	Name string
	Url  string
}

type ComputedResource struct {
	Name   string
	Policy struct {
		Arn string
	}
	Role struct {
		Arn string
	}
	Tags map[string]string
}

type ComputedTemplateData struct {
	AccountId         string
	Region            string
	RegistryRegion    string
	RegistryAccountId string
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
	Registry     ComputedRegistry
	Repository   ComputedRepository
	Resource     ComputedResource
	Resources    ComputedResources
	TemplateData ComputedTemplateData
}

func (c Config) ComputeBuildTime(mfst manifest.BuildTime) (BuildTime, error) {
	buildtime := BuildTime{BuildTime: mfst}
	buildtime.Computed.Registry.Url = c.Registry.Url
	buildtime.Computed.Repository.Solve(c.Registry, c.Repository, c.Git, mfst.Name.Decoded)
	buildtime.Computed.Resource.Solve(c.Account, c.Resource, c.Git, mfst.Name.Decoded)
	buildtime.Computed.Resources.Solve(c.Repository, c.Git, mfst.Resources.Decoded, mfst.Name.Decoded)
	buildtime.Computed.TemplateData.Solve(c.Account, c.Registry)
	return buildtime, nil
}

func (c Config) ComputeDeployTime(mfst manifest.DeployTime) (DeployTime, error) {
	deploytime := DeployTime{DeployTime: mfst}

	origin, err := url.Parse(deploytime.Origin.Decoded)
	if err != nil {
		return DeployTime{}, err
	}

	git := gitlib.DotGit{
		Branch: deploytime.Branch.Decoded,
		Sha:    deploytime.Sha.Decoded,
		Origin: origin,
	}

	deploytime.Computed.Registry.Url = c.Registry.Url
	deploytime.Computed.Repository.Solve(c.Registry, c.Repository, git, deploytime.Name.Decoded)
	deploytime.Computed.Resource.Solve(c.Account, c.Resource, git, deploytime.Name.Decoded)
	deploytime.Computed.Resources.Solve(c.Repository, git, deploytime.Resources.Decoded, deploytime.Name.Decoded)
	deploytime.Computed.TemplateData.Solve(c.Account, c.Registry)
	return deploytime, nil
}

func (r *ComputedRepository) Solve(registry Registry, repository Repository, git gitlib.DotGit, name string) {
	r.Name = filepath.Clean(repository.Namespace + "/" + name)
	r.Url = registry.Url + "/" + r.Name
}

func (r *ComputedResource) Solve(account Account, resource Resource, git gitlib.DotGit, name string) {
	r.Name = resource.Namespace + "-" + git.Branch + "-" + name
	r.Policy.Arn = "arn:aws:iam::" + account.Id + ":policy/" + r.Name
	r.Role.Arn = "arn:aws:iam::" + account.Id + ":role/" + r.Name

	r.Tags = map[string]string{
		"Function": name,
		"Origin":   git.Origin.String(),
		"Branch":   git.Branch,
		"Sha":      git.Sha,
	}
}

func (t *ComputedTemplateData) Solve(account Account, registry Registry) {
	t.AccountId = account.Id
	t.Region = account.Region
	t.RegistryAccountId = registry.Id
	t.RegistryRegion = registry.Region
}

func (resources *ComputedResources) Solve(repository Repository, git gitlib.DotGit, resourcesJson, name string) {
	defaults := ComputedResources{
		EphemeralStorage: 512,
		MemorySize:       128,
		Timeout:          3,
		Http:             true,
		Public:           false,
	}

	if value, exists := os.LookupEnv(EnvOwnerPrefixRoutes); exists {
		if strings.ToLower(value) == "true" {
			defaults.RouteKey = "ANY /" + repository.Namespace + "/" + git.Branch + "/" + name + "/{proxy+}"

		}
	} else {
		noOwner := strings.Split(repository.Namespace, "/")[1:]
		defaults.RouteKey = "ANY /" + strings.Join(noOwner, "/") + "/" + git.Branch + "/" + name + "/{proxy+}"
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
