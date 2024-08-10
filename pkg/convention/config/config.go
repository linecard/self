package config

import (
	"context"
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/aws/aws-lambda-go/events"
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
	EnvAuthType             = "SELF_API_GATEWAY_AUTH_TYPE"
	EnvAuthorizerId         = "SELF_API_GATEWAY_AUTHORIZER_ID"
	EnvSgIds                = "SELF_SECURITY_GROUP_IDS"
	EnvSnIds                = "SELF_SUBNET_IDS"
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

type Selfish struct {
	Path string
	Name string
}

type Config struct {
	Selfish      []Selfish
	Caller       Caller
	Account      Account
	Git          gitlib.DotGit
	Registry     Registry
	Repository   Repository
	Resource     Resource
	ApiGateway   ApiGateway
	Vpc          Vpc
	TemplateData TemplateData
	Version      string
	AwsConfig    aws.Config `json:"-"`
}

func (c Config) Find(buildPath string) (BuildTime, error) {
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

func (c Config) Parse(labels map[string]string) (DeployTime, error) {
	deploytime, err := manifest.Decode(labels, c.TemplateData)
	if err != nil {
		return DeployTime{}, err
	}

	return c.ComputeDeployTime(deploytime)
}

func (c *Config) FromCwd(ctx context.Context, awsConfig aws.Config, ecrc ECRClient, stsc STSClient) (err error) {
	c.AwsConfig = awsConfig

	if err = c.DiscoverGit(ctx); err != nil {
		return
	}

	if err = c.DiscoverCaller(ctx, stsc, awsConfig); err != nil {
		return
	}

	if err = c.DiscoverRegistry(ctx, ecrc, awsConfig); err != nil {
		return
	}

	if err = c.DiscoverGateway(ctx); err != nil {
		return
	}

	if err = c.DiscoverVpc(ctx); err != nil {
		return
	}

	if err = c.DiscoverSelfish(ctx); err != nil {
		return
	}

	nameSpace := strings.TrimSuffix(c.Git.Origin.Path, ".git")
	c.Repository.Namespace = strings.TrimPrefix(nameSpace, "/")

	// temporary backwards compatability envar
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

func (c *Config) FromEvent(ctx context.Context, awsConfig aws.Config, ecrc ECRClient, stsc STSClient, event events.ECRImageActionEvent) (err error) {
	c.AwsConfig = awsConfig

	if err = c.DiscoverCaller(ctx, stsc, awsConfig); err != nil {
		return
	}

	if err = c.DiscoverRegistry(ctx, ecrc, awsConfig); err != nil {
		return
	}

	if err = c.DiscoverGateway(ctx); err != nil {
		return
	}

	if err = c.DiscoverVpc(ctx); err != nil {
		return
	}

	if util.ShaLike(event.Detail.ImageTag) {
		c.Git.Sha = event.Detail.ImageTag
	} else {
		c.Git.Branch = event.Detail.ImageTag
	}

	return
}
