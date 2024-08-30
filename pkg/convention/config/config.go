package config

import (
	"context"
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/linecard/self/internal/gitlib"
	"github.com/linecard/self/internal/util"
	"github.com/linecard/self/pkg/convention/manifest"
)

const (
	EnvGitBranch            = "SELF_BRANCH_OVERRIDE"
	EnvGitSha               = "SELF_SHA_OVERRIDE"
	EnvOwnerPrefixResources = "SELF_PREFIX_RESOURCES_WITH_OWNER"
	EnvOwnerPrefixRoutes    = "SELF_PREFIX_ROUTE_KEY_WITH_OWNER"
	EnvEcrId                = "SELF_ECR_REGISTRY_ID"
	EnvEcrRegion            = "SELF_ECR_REGISTRY_REGION"
	EnvGwId                 = "SELF_API_GATEWAY_ID"
	EnvSgIds                = "SELF_SECURITY_GROUP_IDS"
	EnvSnIds                = "SELF_SUBNET_IDS"
	EnvBusName              = "SELF_SELF_BUS_NAME"
)

//go:embed embedded/*
var embedded embed.FS

type Caller struct {
	Arn string
}

type Account struct {
	Id     string
	Region string
}

type Git struct {
	Origin string
	Path   string
	Branch string
	Sha    string
	Root   string
	Dirty  bool
}

type Registry struct {
	Id     string
	Region string
	Url    string
}

type ApiGateway struct {
	Id *string
}

type Vpc struct {
	SecurityGroupIds []string
	SubnetIds        []string
}

type TemplateData struct {
	AccountId         string
	Region            string
	RegistryRegion    string
	RegistryAccountId string
}

type Repository struct {
	Namespace string
}

type Resource struct {
	Namespace string
}

type Bus struct {
	Name *string
}

type Selfish struct {
	Path string
	Name string
}

type Config struct {
	Selfish      []Selfish
	Caller       Caller
	Account      Account
	Bus          Bus
	Git          gitlib.DotGit
	Registry     Registry
	Repository   Repository
	Resource     Resource
	ApiGateway   ApiGateway
	Vpc          Vpc
	TemplateData TemplateData
	Version      string
}

// Initialize configuration from AWS and local filesystem.
func Stateful(ctx context.Context, awsConfig aws.Config, stsc STSClient, ecrc ECRClient) (c Config, err error) {
	if err = c.FromAws(ctx, awsConfig, stsc, ecrc); err != nil {
		return
	}

	if err = c.FromCwd(ctx); err != nil {
		return
	}

	// This block happens in Stateless as well, needs DRY-ing.
	// Conceptually it's the Computed values of the root Config.
	nameSpace := strings.TrimSuffix(c.Git.Origin.Path, ".git")
	c.Repository.Namespace = strings.TrimPrefix(nameSpace, "/")

	if value, exists := os.LookupEnv(EnvOwnerPrefixResources); exists {
		if strings.ToLower(value) == "true" {
			c.Resource.Namespace = util.DeSlasher(nameSpace)
		}
	} else {
		noOwner := strings.Split(util.DeSlasher(nameSpace), "-")[1:]
		c.Resource.Namespace = strings.Join(noOwner, "-")
	}

	c.TemplateData.AccountId = c.Account.Id
	c.TemplateData.Region = c.Account.Region
	c.TemplateData.RegistryRegion = c.Registry.Region
	c.TemplateData.RegistryAccountId = c.Registry.Id

	return
}

// Initialize configuration from AWS only.
func Stateless(ctx context.Context, awsConfig aws.Config, stsc STSClient, ecrc ECRClient, event Event) (c Config, err error) {
	if err = c.FromAws(ctx, awsConfig, stsc, ecrc); err != nil {
		return
	}

	if err = c.FromEvent(ctx, event); err != nil {
		return
	}

	nameSpace := strings.TrimSuffix(c.Git.Origin.Path, ".git")
	c.Repository.Namespace = strings.TrimPrefix(nameSpace, "/")

	if value, exists := os.LookupEnv(EnvOwnerPrefixResources); exists {
		if strings.ToLower(value) == "true" {
			c.Resource.Namespace = util.DeSlasher(nameSpace)
		}
	} else {
		noOwner := strings.Split(util.DeSlasher(nameSpace), "-")[1:]
		c.Resource.Namespace = strings.Join(noOwner, "-")
	}

	c.TemplateData.AccountId = c.Account.Id
	c.TemplateData.Region = c.Account.Region
	c.TemplateData.RegistryRegion = c.Registry.Region
	c.TemplateData.RegistryAccountId = c.Registry.Id

	return
}

// Generate buildtime configuration from a selfish path.
func (c Config) BuildTime(buildPath string) (BuildTime, error) {
	absPath, err := filepath.Abs(buildPath)
	if err != nil {
		return BuildTime{}, err
	}

	for _, s := range c.Selfish {
		if s.Path == absPath {
			buildtime, err := manifest.Encode(absPath, c.Git)
			if err != nil {
				return BuildTime{}, err
			}
			return c.ComputeBuildTime(buildtime)
		}
	}

	return BuildTime{}, fmt.Errorf("no selfish found for %s", absPath)
}

// Generate deploytime configuration from release labels.
func (c Config) DeployTime(labels map[string]string) (DeployTime, error) {
	deploytime, err := manifest.Decode(labels, c.TemplateData)
	if err != nil {
		return DeployTime{}, err
	}

	return c.ComputeDeployTime(deploytime)
}
