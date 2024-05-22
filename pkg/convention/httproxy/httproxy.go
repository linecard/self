package httproxy

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/linecard/self/internal/labelgun"
	"github.com/linecard/self/pkg/convention/config"
	"github.com/linecard/self/pkg/convention/deployment"

	"github.com/aws/aws-sdk-go-v2/service/apigatewayv2"
	"github.com/aws/aws-sdk-go-v2/service/apigatewayv2/types"
	dockerTypes "github.com/docker/docker/api/types"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
)

type GatewayService interface {
	GetApi(ctx context.Context, apiId string) (*apigatewayv2.GetApiOutput, error)
	PutIntegration(ctx context.Context, apiId, lambdaArn, routeKey string) (*apigatewayv2.GetIntegrationOutput, error)
	PutRoute(ctx context.Context, apiId, integrationId, routeKey string) (*apigatewayv2.GetRouteOutput, error)
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

	r, err := d.FetchRelease(ctx, c.Service.Registry, c.Config.Registry.Id)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		return err
	}

	if !labelgun.HasLabel(c.Config.Label.Resources, r.Config.Labels) {
		if err := c.Unmount(ctx, d); err != nil {
			return err
		}
		return nil
	}

	resources := struct {
		Http bool `json:"http"`
	}{
		Http: false,
	}

	resourcesTemplate, err := labelgun.DecodeLabel(c.Config.Label.Resources, r.Config.Labels)
	if err != nil {
		return err
	}

	resourcesDocument, err := c.Config.Template(resourcesTemplate)
	if err != nil {
		return err
	}

	if err := json.Unmarshal([]byte(resourcesDocument), &resources); err != nil {
		return err
	}

	if resources.Http {
		if err := c.Mount(ctx, d, namespace); err != nil {
			return err
		}
		return nil
	}

	if err := c.Unmount(ctx, d); err != nil {
		return err
	}

	return nil
}

func (c Convention) Mount(ctx context.Context, d deployment.Deployment, namespace string) error {
	gw, err := c.Service.Gateway.GetApi(ctx, c.Config.Httproxy.ApiId)
	if err != nil {
		return err
	}

	routeKey := c.Config.RouteKey(namespace)

	// This should also be enforced in IAM policy with a conditional.
	if _, exist := gw.Tags["SelfManaged"]; !exist {
		return fmt.Errorf("api gateway %s does not have SelfManaged tag", *gw.ApiId)
	}

	integration, err := c.Service.Gateway.PutIntegration(ctx, c.Config.Httproxy.ApiId, *d.Configuration.FunctionArn, routeKey)
	if err != nil {
		return err
	}

	_, err = c.Service.Gateway.PutRoute(ctx, c.Config.Httproxy.ApiId, *integration.IntegrationId, routeKey)
	if err != nil {
		return err
	}

	err = c.Service.Gateway.PutLambdaPermission(ctx, c.Config.Httproxy.ApiId, *d.Configuration.FunctionArn, routeKey)
	if err != nil {
		return err
	}

	return nil
}

func (c Convention) Unmount(ctx context.Context, d deployment.Deployment) error {
	ctx, span := otel.Tracer("").Start(ctx, "httproxy.Unmount")
	defer span.End()

	gw, err := c.Service.Gateway.GetApi(ctx, c.Config.Httproxy.ApiId)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		return err
	}

	// This should also be enforced in IAM policy with a conditional.
	if _, exist := gw.Tags["SelfManaged"]; !exist {
		err := fmt.Errorf("api gateway %s does not have SelfManaged tag", *gw.ApiId)
		span.SetStatus(codes.Error, err.Error())
		return err
	}

	routes, err := c.Service.Gateway.GetRoutesByFunctionArn(ctx, c.Config.Httproxy.ApiId, *d.Configuration.FunctionArn)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		return err
	}

	for _, route := range routes {
		err = c.Service.Gateway.DeleteRoute(ctx, c.Config.Httproxy.ApiId, route)
		if err != nil {
			span.SetStatus(codes.Error, err.Error())
			return err
		}

		err = c.Service.Gateway.DeleteLambdaPermission(ctx, *d.Configuration.FunctionArn, route)
		if err != nil {
			span.SetStatus(codes.Error, err.Error())
			return err
		}

		err = c.Service.Gateway.DeleteIntegration(ctx, c.Config.Httproxy.ApiId, route)
		if err != nil {
			span.SetStatus(codes.Error, err.Error())
			return err
		}
	}

	return nil
}

func (c Convention) ListRoutes(ctx context.Context, d deployment.Deployment) ([]types.Route, error) {
	return c.Service.Gateway.GetRoutesByFunctionArn(ctx, c.Config.Httproxy.ApiId, *d.Configuration.FunctionArn)
}
