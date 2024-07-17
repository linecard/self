package config

import (
	"context"
	"embed"
	"fmt"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/linecard/self/internal/util"
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

type Config struct {
	Functions    []ReleaseSchema
	Caller       Caller
	Account      Account
	Git          Git
	Registry     Registry
	ApiGateway   ApiGateway
	Vpc          Vpc
	TemplateData TemplateData
	Version      string
}

func (c *Config) Function(name string) (ReleaseSchema, error) {
	for _, rc := range c.Functions {
		if rc.Name.Content == name {
			return rc, nil
		}
	}

	return ReleaseSchema{}, fmt.Errorf("function %s not found", name)
}

func (c *Config) FromCwd(ctx context.Context, awsConfig aws.Config, ecrc ECRClient, stsc STSClient) (err error) {
	if err := c.DiscoverGit(ctx); err != nil {
		return err
	}

	if err := c.DiscoverCaller(ctx, stsc, awsConfig); err != nil {
		return err
	}

	if err := c.DiscoverRegistry(ctx, envEcrId, envEcrRegion, ecrc, awsConfig); err != nil {
		return err
	}

	if err := c.DiscoverGateway(ctx, envGwId); err != nil {
		return err
	}

	if err := c.DiscoverVpc(ctx, envSgIds, envSnIds); err != nil {
		return err
	}

	if err := c.DiscoverFunctions(ctx); err != nil {
		return err
	}

	c.TemplateData = TemplateData{
		AccountId:         c.Account.Id,
		Region:            c.Account.Region,
		RegistryRegion:    c.Registry.Region,
		RegistryAccountId: c.Registry.Id,
	}

	return nil
}

func (c *Config) FromEvent(ctx context.Context, awsConfig aws.Config, ecrc ECRClient, stsc STSClient, event events.ECRImageActionEvent) (err error) {
	if err := c.DiscoverCaller(ctx, stsc, awsConfig); err != nil {
		return err
	}

	if err := c.DiscoverRegistry(ctx, envEcrId, envEcrRegion, ecrc, awsConfig); err != nil {
		return err
	}

	if err := c.DiscoverGateway(ctx, envGwId); err != nil {
		return err
	}

	if err := c.DiscoverVpc(ctx, envSgIds, envSnIds); err != nil {
		return err
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

	return nil
}
