package gateway

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/apigatewayv2"
	"github.com/aws/aws-sdk-go-v2/service/apigatewayv2/types"
	"github.com/aws/aws-sdk-go-v2/service/lambda"
	"github.com/aws/smithy-go"
	"github.com/linecard/self/internal/util"
)

type ApiGatewayV2Client interface {
	CreateIntegration(ctx context.Context, params *apigatewayv2.CreateIntegrationInput, optFns ...func(*apigatewayv2.Options)) (*apigatewayv2.CreateIntegrationOutput, error)
	DeleteIntegration(ctx context.Context, params *apigatewayv2.DeleteIntegrationInput, optFns ...func(*apigatewayv2.Options)) (*apigatewayv2.DeleteIntegrationOutput, error)
	GetIntegrations(ctx context.Context, params *apigatewayv2.GetIntegrationsInput, optFns ...func(*apigatewayv2.Options)) (*apigatewayv2.GetIntegrationsOutput, error)
	GetIntegration(ctx context.Context, params *apigatewayv2.GetIntegrationInput, optFns ...func(*apigatewayv2.Options)) (*apigatewayv2.GetIntegrationOutput, error)
	UpdateIntegration(ctx context.Context, params *apigatewayv2.UpdateIntegrationInput, optFns ...func(*apigatewayv2.Options)) (*apigatewayv2.UpdateIntegrationOutput, error)
	CreateRoute(ctx context.Context, params *apigatewayv2.CreateRouteInput, optFns ...func(*apigatewayv2.Options)) (*apigatewayv2.CreateRouteOutput, error)
	GetRoutes(ctx context.Context, params *apigatewayv2.GetRoutesInput, optFns ...func(*apigatewayv2.Options)) (*apigatewayv2.GetRoutesOutput, error)
	GetRoute(ctx context.Context, params *apigatewayv2.GetRouteInput, optFns ...func(*apigatewayv2.Options)) (*apigatewayv2.GetRouteOutput, error)
	DeleteRoute(ctx context.Context, params *apigatewayv2.DeleteRouteInput, optFns ...func(*apigatewayv2.Options)) (*apigatewayv2.DeleteRouteOutput, error)
	UpdateRoute(ctx context.Context, params *apigatewayv2.UpdateRouteInput, optFns ...func(*apigatewayv2.Options)) (*apigatewayv2.UpdateRouteOutput, error)
	GetApi(ctx context.Context, params *apigatewayv2.GetApiInput, optFns ...func(*apigatewayv2.Options)) (*apigatewayv2.GetApiOutput, error)
}

type LambdaClient interface {
	AddPermission(ctx context.Context, params *lambda.AddPermissionInput, optFns ...func(*lambda.Options)) (*lambda.AddPermissionOutput, error)
	RemovePermission(ctx context.Context, params *lambda.RemovePermissionInput, optFns ...func(*lambda.Options)) (*lambda.RemovePermissionOutput, error)
}

type Client struct {
	Gw     ApiGatewayV2Client
	Lambda LambdaClient
}

type Service struct {
	Client Client
}

func FromClients(gwc ApiGatewayV2Client, lmc LambdaClient) Service {
	return Service{
		Client: Client{
			Gw:     gwc,
			Lambda: lmc,
		},
	}
}

func (s Service) PutIntegration(ctx context.Context, apiId, lambdaArn, routeKey string) (*apigatewayv2.GetIntegrationOutput, error) {
	integrations, err := s.Client.Gw.GetIntegrations(ctx, &apigatewayv2.GetIntegrationsInput{
		ApiId: aws.String(apiId),
	})

	if err != nil {
		return nil, err
	}

	forwardedForPrefix := strings.Split(routeKey, " ")[1]
	forwardedForPrefix = strings.Replace(forwardedForPrefix, "/{proxy+}", "", 1)

	for _, integration := range integrations.Items {
		if *integration.IntegrationUri == lambdaArn {
			updated, err := s.Client.Gw.UpdateIntegration(ctx, &apigatewayv2.UpdateIntegrationInput{
				ApiId:                aws.String(apiId),
				IntegrationId:        integration.IntegrationId,
				IntegrationUri:       aws.String(lambdaArn),
				PayloadFormatVersion: aws.String("2.0"),
				RequestParameters: map[string]string{
					"overwrite:path":                      "/$request.path.proxy",
					"overwrite:header.X-Forwarded-Prefix": forwardedForPrefix,
				},
			})

			if err != nil {
				return nil, err
			}

			return s.Client.Gw.GetIntegration(ctx, &apigatewayv2.GetIntegrationInput{
				ApiId:         aws.String(apiId),
				IntegrationId: updated.IntegrationId,
			})
		}
	}

	created, err := s.Client.Gw.CreateIntegration(ctx, &apigatewayv2.CreateIntegrationInput{
		ApiId:                aws.String(apiId),
		IntegrationType:      types.IntegrationTypeAwsProxy,
		IntegrationUri:       aws.String(lambdaArn),
		PayloadFormatVersion: aws.String("2.0"),
		RequestParameters: map[string]string{
			"overwrite:path":                      "/$request.path.proxy",
			"overwrite:header.X-Forwarded-Prefix": forwardedForPrefix,
		},
	})

	if err != nil {
		return nil, err
	}

	return s.Client.Gw.GetIntegration(ctx, &apigatewayv2.GetIntegrationInput{
		ApiId:         aws.String(apiId),
		IntegrationId: created.IntegrationId,
	})
}

func (s Service) PutRoute(ctx context.Context, apiId, integrationId, routeKey string) (*apigatewayv2.GetRouteOutput, error) {
	routes, err := s.Client.Gw.GetRoutes(ctx, &apigatewayv2.GetRoutesInput{
		ApiId: aws.String(apiId),
	})

	if err != nil {
		return nil, err
	}

	for _, route := range routes.Items {
		if *route.RouteKey == routeKey {
			updated, err := s.Client.Gw.UpdateRoute(ctx, &apigatewayv2.UpdateRouteInput{
				ApiId:             aws.String(apiId),
				RouteId:           route.RouteId,
				RouteKey:          aws.String(routeKey),
				Target:            aws.String(fmt.Sprintf("integrations/%s", integrationId)),
				AuthorizationType: types.AuthorizationTypeAwsIam,
			})

			if err != nil {
				return nil, err
			}

			return s.Client.Gw.GetRoute(ctx, &apigatewayv2.GetRouteInput{
				ApiId:   aws.String(apiId),
				RouteId: updated.RouteId,
			})
		}
	}

	created, err := s.Client.Gw.CreateRoute(ctx, &apigatewayv2.CreateRouteInput{
		ApiId:             aws.String(apiId),
		RouteKey:          aws.String(routeKey),
		Target:            aws.String(fmt.Sprintf("integrations/%s", integrationId)),
		AuthorizationType: types.AuthorizationTypeAwsIam,
	})

	if err != nil {
		return nil, err
	}

	return s.Client.Gw.GetRoute(ctx, &apigatewayv2.GetRouteInput{
		ApiId:   aws.String(apiId),
		RouteId: created.RouteId,
	})
}

func (s Service) PutLambdaPermission(ctx context.Context, apiId, lambdaArn, routeKey string) error {
	var apiErr smithy.APIError

	accountId := strings.Split(lambdaArn, ":")[4]
	region := strings.Split(lambdaArn, ":")[3]
	routePrefix := strings.Split(routeKey, " ")[1]
	statementId := strings.TrimPrefix("-", util.DeSlasher(routePrefix)+"-api-gw")

	_, err := s.Client.Lambda.AddPermission(ctx, &lambda.AddPermissionInput{
		Action:       aws.String("lambda:InvokeFunction"),
		FunctionName: aws.String(lambdaArn),
		Principal:    aws.String("apigateway.amazonaws.com"),
		SourceArn:    aws.String("arn:aws:execute-api:" + region + ":" + accountId + ":" + apiId + "/*/*" + routePrefix),
		StatementId:  aws.String(statementId),
	})

	if errors.As(err, &apiErr) && apiErr.ErrorCode() == "ResourceConflictException" {
		return nil
	}

	if err != nil {
		return err
	}

	return nil
}

func (s Service) DeleteIntegration(ctx context.Context, apiId string, route types.Route) error {
	var apiErr smithy.APIError
	var integrationId string

	fmt.Sscanf(*route.Target, "integrations/%s", &integrationId)

	_, err := s.Client.Gw.DeleteIntegration(ctx, &apigatewayv2.DeleteIntegrationInput{
		ApiId:         aws.String(apiId),
		IntegrationId: aws.String(integrationId),
	})

	if errors.As(err, &apiErr) && apiErr.ErrorCode() == "NotFoundException" {
		return fmt.Errorf("integration %s not found under api %s", integrationId, apiId)
	}

	if err != nil {
		return err
	}

	return nil
}

func (s Service) DeleteRoute(ctx context.Context, apiId string, route types.Route) error {
	var apiErr smithy.APIError

	_, err := s.Client.Gw.DeleteRoute(ctx, &apigatewayv2.DeleteRouteInput{
		ApiId:   aws.String(apiId),
		RouteId: aws.String(*route.RouteId),
	})

	if errors.As(err, &apiErr) && apiErr.ErrorCode() == "NotFoundException" {
		return fmt.Errorf("route %s not found under api %s", *route.RouteKey, apiId)
	}

	if err != nil {
		return err
	}

	return nil
}

func (s Service) DeleteLambdaPermission(ctx context.Context, lambdaArn string, route types.Route) error {
	var apiErr smithy.APIError
	routePrefix := strings.Split(*route.RouteKey, " ")[1]
	statementId := strings.TrimPrefix("-", util.DeSlasher(routePrefix)+"-api-gw")

	_, err := s.Client.Lambda.RemovePermission(ctx, &lambda.RemovePermissionInput{
		FunctionName: aws.String(lambdaArn),
		StatementId:  aws.String(statementId),
	})

	if errors.As(err, &apiErr) && apiErr.ErrorCode() == "ResourceNotFoundException" {
		return fmt.Errorf("permission for route %s not found", routePrefix)
	}

	if err != nil {
		return err
	}

	return nil
}

func (s Service) GetApi(ctx context.Context, apiId string) (*apigatewayv2.GetApiOutput, error) {
	return s.Client.Gw.GetApi(ctx, &apigatewayv2.GetApiInput{
		ApiId: aws.String(apiId),
	})
}

func (s Service) GetRouteByRouteKey(ctx context.Context, apiId, routeKey string) (types.Route, error) {
	var matches []types.Route

	routes, err := s.Client.Gw.GetRoutes(ctx, &apigatewayv2.GetRoutesInput{
		ApiId: aws.String(apiId),
	})

	if err != nil {
		return types.Route{}, err
	}

	for _, route := range routes.Items {
		if *route.RouteKey == routeKey {
			matches = append(matches, route)
		}
	}

	if len(matches) == 0 {
		return types.Route{}, nil
	}

	if len(matches) > 1 {
		return types.Route{}, fmt.Errorf("multiple routes found under api %s with route key %s", apiId, routeKey)
	}

	return matches[0], nil
}

func (s Service) GetRoutesByFunctionArn(ctx context.Context, apiId, functionArn string) ([]types.Route, error) {
	var associatedIntegrations []types.Integration
	var integratedRoutes []types.Route

	routes, err := s.Client.Gw.GetRoutes(ctx, &apigatewayv2.GetRoutesInput{
		ApiId: aws.String(apiId),
	})

	if err != nil {
		return nil, err
	}

	integrations, err := s.Client.Gw.GetIntegrations(ctx, &apigatewayv2.GetIntegrationsInput{
		ApiId: aws.String(apiId),
	})

	if err != nil {
		return nil, err
	}

	for _, integration := range integrations.Items {
		if *integration.IntegrationUri == functionArn {
			associatedIntegrations = append(associatedIntegrations, integration)
		}
	}

	for _, integration := range associatedIntegrations {
		for _, route := range routes.Items {
			routeIntegrationId := strings.TrimPrefix(*route.Target, "integrations/")
			if routeIntegrationId == *integration.IntegrationId {
				integratedRoutes = append(integratedRoutes, route)
			}
		}
	}

	return integratedRoutes, nil
}
