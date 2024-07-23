package config

import (
	"context"
	"embed"
	"fmt"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/linecard/self/internal/gitlib"
	"github.com/linecard/self/internal/util"
	"github.com/linecard/self/pkg/convention/manifest"
)

const (
	envEcrId     = "AWS_ECR_REGISTRY_ID"
	envEcrRegion = "AWS_ECR_REGISTRY_REGION"
	envGwId      = "AWS_API_GATEWAY_ID"
	envSgIds     = "AWS_SECURITY_GROUP_IDS"
	envSnIds     = "AWS_SUBNET_IDS"
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
	ApiGateway   ApiGateway
	Vpc          Vpc
	TemplateData TemplateData
	Version      string
}

func (c Config) Find(path string) (buildtime manifest.BuildTime, computed Computed, err error) {
	for _, s := range c.Selfish {
		if s.Path == path {
			if buildtime, err = manifest.Encode(path, c.Git); err != nil {
				return
			}

			return c.SolveBuildTime(buildtime)
		}
	}

	return buildtime, computed, fmt.Errorf("%s does not appear to be selfish", path)
}

func (c Config) Parse(labels map[string]string) (deploytime manifest.DeployTime, computed Computed, err error) {
	deploytime, err = manifest.Decode(labels, c.TemplateData)
	if err != nil {
		return
	}

	return c.SolveDeployTime(deploytime)
}

func (c *Config) FromCwd(ctx context.Context, awsConfig aws.Config, ecrc ECRClient, stsc STSClient) (err error) {
	if err = c.DiscoverGit(ctx); err != nil {
		return
	}

	if err = c.DiscoverCaller(ctx, stsc, awsConfig); err != nil {
		return
	}

	if err = c.DiscoverRegistry(ctx, envEcrId, envEcrRegion, ecrc, awsConfig); err != nil {
		return
	}

	if err = c.DiscoverGateway(ctx, envGwId); err != nil {
		return
	}

	if err = c.DiscoverVpc(ctx, envSgIds, envSnIds); err != nil {
		return
	}

	if err = c.DiscoverSelfish(ctx); err != nil {
		return
	}

	c.TemplateData = TemplateData{
		AccountId:         c.Account.Id,
		Region:            c.Account.Region,
		RegistryRegion:    c.Registry.Region,
		RegistryAccountId: c.Registry.Id,
	}

	return
}

func (c *Config) FromEvent(ctx context.Context, awsConfig aws.Config, ecrc ECRClient, stsc STSClient, event events.ECRImageActionEvent) (err error) {
	if err = c.DiscoverCaller(ctx, stsc, awsConfig); err != nil {
		return
	}

	if err = c.DiscoverRegistry(ctx, envEcrId, envEcrRegion, ecrc, awsConfig); err != nil {
		return
	}

	if err = c.DiscoverGateway(ctx, envGwId); err != nil {
		return
	}

	if err = c.DiscoverVpc(ctx, envSgIds, envSnIds); err != nil {
		return
	}

	if util.ShaLike(event.Detail.ImageTag) {
		c.Git.Sha = event.Detail.ImageTag
	} else {
		c.Git.Branch = event.Detail.ImageTag
	}

	c.TemplateData = TemplateData{
		AccountId:         c.Account.Id,
		Region:            c.Account.Region,
		RegistryRegion:    c.Registry.Region,
		RegistryAccountId: c.Registry.Id,
	}

	return
}
