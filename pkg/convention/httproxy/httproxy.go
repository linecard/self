package httproxy

import (
	"context"
	"encoding/json"

	"github.com/linecard/self/pkg/convention/config"
	"github.com/linecard/self/pkg/convention/deployment"

	"github.com/aws/aws-sdk-go-v2/service/apigatewayv2"
	"github.com/aws/aws-sdk-go-v2/service/apigatewayv2/types"
	dockerTypes "github.com/docker/docker/api/types"
	"github.com/rs/zerolog/log"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
)

type GatewayService interface {
	GetApi(ctx context.Context, apiId string) (*apigatewayv2.GetApiOutput, error)
	GetApis(ctx context.Context) (*apigatewayv2.GetApisOutput, error)
	PutIntegration(ctx context.Context, apiId, lambdaArn, routeKey string) (*apigatewayv2.GetIntegrationOutput, error)
	PutRoute(ctx context.Context, apiId, integrationId, routeKey string, awsAuth bool) (*apigatewayv2.GetRouteOutput, error)
	PutLambdaPermission(ctx context.Context, apiId, lambdaArn, routeKey string) error
	DeleteIntegration(ctx context.Context, apiId string, route types.Route) error
	DeleteRoute(ctx context.Context, apiId string, route types.Route) error
	DeleteLambdaPermission(ctx context.Context, lambdaArn string, route types.Route) error
	GetRouteByRouteKey(ctx context.Context, apiId, routeKey string) (types.Route, error)
	GetRoutesByFunctionArn(ctx context.Context, apiId, functionArn string) ([]types.Route, error)
}

type RegistryService interface {
	InspectByDigest(ctx context.Context, registryId, repository, digest string) (dockerTypes.ImageInspect, error)
}

type Services struct {
	Gateway  GatewayService
	Registry RegistryService
}

type Convention struct {
	Config  config.Config
	Service Services
}

func FromServices(c config.Config, g GatewayService, r RegistryService) Convention {
	return Convention{
		Config: c,
		Service: Services{
			Gateway:  g,
			Registry: r,
		},
	}
}

func (c Convention) Converge(ctx context.Context, d deployment.Deployment, namespace string) error {
	ctx, span := otel.Tracer("").Start(ctx, "httproxy.Converge")
	defer span.End()

	if c.Config.ApiGateway.Id == nil {
		log.Info().Msg("no api gateway defined, clearing associated proxy routes")
		return c.Unmount(ctx, d)
	}

	release, err := d.FetchRelease(ctx, c.Service.Registry, c.Config.Registry.Id)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		return err
	}

	labels, err := c.Config.Labels.Decode(release.Config.Labels)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		return err
	}

	for k, v := range labels {
		templatedValue, err := c.Config.Template(v)
		if err != nil {
			span.SetStatus(codes.Error, err.Error())
			return err
		}

		labels[k] = templatedValue
	}

	if _, exists := labels[c.Config.Labels.Resources.Key]; !exists {
		log.Info().Msgf("no label %s defined, unmounting any associated proxy routes targeting %s", c.Config.Labels.Resources.Key, *d.Configuration.FunctionArn)
		if err := c.Unmount(ctx, d); err != nil {
			return err
		}

		return nil
	}

	resources := struct {
		Http   bool `json:"http"`
		Public bool `json:"public"`
	}{
		Http:   false,
		Public: false,
	}

	if err := json.Unmarshal([]byte(labels[c.Config.Labels.Resources.Key]), &resources); err != nil {
		span.SetStatus(codes.Error, err.Error())
		return err
	}

	if !resources.Http {
		return c.Unmount(ctx, d)
	}

	c.Mount(ctx, d, namespace, !resources.Public)

	return nil
}

func (c Convention) Mount(ctx context.Context, d deployment.Deployment, namespace string, awsAuth bool) error {
	if c.Config.ApiGateway.Id == nil {
		log.Info().Msg("no api gateway defined, skipping httproxy mount")
		return nil
	}

	routeKey := c.Config.RouteKey(namespace)

	integration, err := c.Service.Gateway.PutIntegration(ctx, *c.Config.ApiGateway.Id, *d.Configuration.FunctionArn, routeKey)
	if err != nil {
		return err
	}

	_, err = c.Service.Gateway.PutRoute(ctx, *c.Config.ApiGateway.Id, *integration.IntegrationId, routeKey, awsAuth)
	if err != nil {
		return err
	}

	err = c.Service.Gateway.PutLambdaPermission(ctx, *c.Config.ApiGateway.Id, *d.Configuration.FunctionArn, routeKey)
	if err != nil {
		return err
	}

	return nil
}

func (c Convention) Unmount(ctx context.Context, d deployment.Deployment) error {
	ctx, span := otel.Tracer("").Start(ctx, "httproxy.Unmount")
	defer span.End()

	apis, err := c.Service.Gateway.GetApis(ctx)
	if err != nil {
		return err
	}

	for _, api := range apis.Items {
		routes, err := c.Service.Gateway.GetRoutesByFunctionArn(ctx, *api.ApiId, *d.Configuration.FunctionArn)
		if err != nil {
			return err
		}

		for _, route := range routes {
			err = c.Service.Gateway.DeleteRoute(ctx, *api.ApiId, route)
			if err != nil {
				return err
			}

			err = c.Service.Gateway.DeleteLambdaPermission(ctx, *d.Configuration.FunctionArn, route)
			if err != nil {
				return err
			}

			err = c.Service.Gateway.DeleteIntegration(ctx, *api.ApiId, route)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (c Convention) ListRoutes(ctx context.Context, d deployment.Deployment) ([]types.Route, error) {
	return c.Service.Gateway.GetRoutesByFunctionArn(ctx, *c.Config.ApiGateway.Id, *d.Configuration.FunctionArn)
}

// for view layer only
func (c Convention) UnsafeListRoutes(ctx context.Context, d deployment.Deployment) ([]types.Route, error) {
	if c.Config.ApiGateway.Id == nil {
		return []types.Route{}, nil
	}
	return c.ListRoutes(ctx, d)
}
