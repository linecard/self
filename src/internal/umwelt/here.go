package umwelt

import (
	"context"
	"fmt"
	"net/url"
	"path"
	"path/filepath"

	"github.com/linecard/self/internal/gitlib"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/apigatewayv2"
	"github.com/aws/aws-sdk-go-v2/service/ecr"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"github.com/linecard/self/internal/util"
)

type STSClient interface {
	GetCallerIdentity(ctx context.Context, params *sts.GetCallerIdentityInput, optFns ...func(*sts.Options)) (*sts.GetCallerIdentityOutput, error)
	AssumeRole(ctx context.Context, params *sts.AssumeRoleInput, optFns ...func(*sts.Options)) (*sts.AssumeRoleOutput, error)
}

type ECRClient interface {
	DescribeRegistry(ctx context.Context, params *ecr.DescribeRegistryInput, optFns ...func(*ecr.Options)) (*ecr.DescribeRegistryOutput, error)
}

type ApiGatewayClient interface {
	GetApis(ctx context.Context, params *apigatewayv2.GetApisInput, optFns ...func(*apigatewayv2.Options)) (*apigatewayv2.GetApisOutput, error)
}

type ThisGit struct {
	Branch string
	Sha    string
	Root   string
	Origin *url.URL
	Dirty  bool
}

type ThisRegistry struct {
	Id     string
	Region string
}

type ThisApiGateway struct {
	Id string
}

type ThisCaller struct {
	Id      string
	Arn     string
	Account string
	Region  string
}

type ThisFunction struct {
	Name string
	Path string
}

type Here struct {
	Caller     ThisCaller
	Git        gitlib.DotGit
	Registry   ThisRegistry
	ApiGateway ThisApiGateway
	Function   *ThisFunction
	Functions  []ThisFunction
}

func FromCwd(ctx context.Context, cwd string, git gitlib.DotGit, awsConfig aws.Config, ecrc ECRClient, gwc ApiGatewayClient, stsc STSClient) (here Here, err error) {
	// Caller
	whoAmI := &sts.GetCallerIdentityInput{}
	caller, err := stsc.GetCallerIdentity(ctx, whoAmI)
	if err != nil {
		return here, err
	}

	here.Caller.Id = *caller.UserId
	here.Caller.Arn = *caller.Arn
	here.Caller.Account = *caller.Account
	here.Caller.Region = awsConfig.Region

	//Git
	here.Git = git

	// Function
	here.Function = Selfish(cwd)
	here.Functions = SelfDiscovery(here.Git.Root)

	// Registry
	here.Registry.Region = GetRegistryRegion("AWS_ECR_REGION", awsConfig)
	if here.Registry.Id, err = GetRegistryId(ctx, "AWS_ECR_REGISTRY_ID", ecrc); err != nil {
		return here, err
	}

	// Gateway
	if here.ApiGateway.Id, err = GetApiGatewayId("AWS_API_GATEWAY_ID", gwc); err != nil {
		return here, err
	}

	return here, nil
}

func FromEvent(ctx context.Context, event events.ECRImageActionEvent, awsConfig aws.Config, ecrc ECRClient, gwc ApiGatewayClient, stsc STSClient) (here Here, err error) {
	// Caller
	whoAmI := &sts.GetCallerIdentityInput{}
	whoIAm, err := stsc.GetCallerIdentity(ctx, whoAmI)
	if err != nil {
		return here, err
	}

	here.Caller.Id = *whoIAm.UserId
	here.Caller.Arn = *whoIAm.Arn
	here.Caller.Account = *whoIAm.Account
	here.Caller.Region = awsConfig.Region

	// Git
	mockedHostOrigin := fmt.Sprintf("%s/%s", "https://github.com", path.Dir(event.Detail.RepositoryName))
	if here.Git.Origin, err = url.Parse(mockedHostOrigin); err != nil {
		return here, err
	}

	if util.ShaLike(event.Detail.ImageTag) {
		here.Git.Sha = event.Detail.ImageTag
	} else {
		here.Git.Branch = event.Detail.ImageTag
	}

	// Function
	here.Function = &ThisFunction{
		Name: filepath.Base(event.Detail.RepositoryName),
	}

	// Registry
	here.Registry.Region = GetRegistryRegion("AWS_ECR_REGION", awsConfig)
	if here.Registry.Id, err = GetRegistryId(ctx, "AWS_ECR_REGISTRY_ID", ecrc); err != nil {
		return here, err
	}

	// Gateway
	if here.ApiGateway.Id, err = GetApiGatewayId("AWS_API_GATEWAY_ID", gwc); err != nil {
		return here, err
	}

	return here, nil
}
