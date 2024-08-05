package sdk

import (
	"context"

	// config
	"github.com/linecard/self/pkg/convention/config"

	// services
	"github.com/linecard/self/pkg/service/docker"
	"github.com/linecard/self/pkg/service/event"
	"github.com/linecard/self/pkg/service/function"
	"github.com/linecard/self/pkg/service/gateway"
	"github.com/linecard/self/pkg/service/registry"

	// conventions
	"github.com/linecard/self/pkg/convention/account"
	"github.com/linecard/self/pkg/convention/bus"
	"github.com/linecard/self/pkg/convention/deployment"
	"github.com/linecard/self/pkg/convention/httproxy"
	"github.com/linecard/self/pkg/convention/release"
	"github.com/linecard/self/pkg/convention/runtime"

	// clients
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/apigatewayv2"
	"github.com/aws/aws-sdk-go-v2/service/ecr"
	"github.com/aws/aws-sdk-go-v2/service/eventbridge"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	"github.com/aws/aws-sdk-go-v2/service/lambda"
	"github.com/aws/aws-sdk-go-v2/service/sts"
)

type Clients struct {
	StsClient          *sts.Client
	EcrClient          *ecr.Client
	LambdaClient       *lambda.Client
	IamClient          *iam.Client
	EventBridgeClient  *eventbridge.Client
	ApiGatewayV2Client *apigatewayv2.Client
}

type Services struct {
	Docker   docker.Service
	Registry registry.Service
	Function function.Service
	Event    event.Service
	Gateway  gateway.Service
}

type Conventions struct {
	Account      account.Convention
	Runtime      runtime.Convention
	Release      release.Convention
	Deployment   deployment.Convention
	Subscription bus.Convention
	Httproxy     httproxy.Convention
}

type API struct {
	Conventions
	Config config.Config
}

func Init(ctx context.Context, awsConfig aws.Config, config config.Config) (API, error) {
	clients, err := InitClients(ctx, awsConfig)
	if err != nil {
		return API{}, err
	}

	services, err := InitServices(ctx, clients)
	if err != nil {
		return API{}, err
	}

	conventions, err := InitConventions(ctx, config, services)
	if err != nil {
		return API{}, err
	}

	return API{
		Conventions: conventions,
		Config:      config,
	}, nil
}

func InitConventions(ctx context.Context, config config.Config, services Services) (Conventions, error) {
	return Conventions{
		Account:      account.FromServices(config, services.Docker, services.Registry),
		Runtime:      runtime.FromServices(config, services.Docker),
		Release:      release.FromServices(config, services.Registry, services.Docker),
		Deployment:   deployment.FromServices(config, services.Function, services.Registry),
		Subscription: bus.FromServices(config, services.Registry, services.Event),
		Httproxy:     httproxy.FromServices(config, services.Gateway, services.Registry),
	}, nil
}

func InitServices(ctx context.Context, clients Clients) (Services, error) {
	docker, err := docker.FromPath(ctx)
	if err != nil {
		return Services{}, err
	}

	return Services{
		Docker:   docker,
		Registry: registry.FromClients(clients.EcrClient),
		Function: function.FromClients(clients.LambdaClient, clients.IamClient),
		Event:    event.FromClients(clients.EventBridgeClient, clients.LambdaClient),
		Gateway:  gateway.FromClients(clients.ApiGatewayV2Client, clients.LambdaClient),
	}, nil
}

func InitClients(ctx context.Context, awsConfig aws.Config) (Clients, error) {
	return Clients{
		StsClient:          sts.NewFromConfig(awsConfig),
		EcrClient:          ecr.NewFromConfig(awsConfig),
		LambdaClient:       lambda.NewFromConfig(awsConfig),
		IamClient:          iam.NewFromConfig(awsConfig),
		EventBridgeClient:  eventbridge.NewFromConfig(awsConfig),
		ApiGatewayV2Client: apigatewayv2.NewFromConfig(awsConfig),
	}, nil
}
