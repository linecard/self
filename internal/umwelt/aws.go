package umwelt

import (
	"context"
	"fmt"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/apigatewayv2"
	"github.com/aws/aws-sdk-go-v2/service/ecr"

	"github.com/rs/zerolog/log"
)

func GetRegistryId(ctx context.Context, envar string, fallback ECRClient) (string, error) {
	var id string
	var exists bool

	if id, exists = os.LookupEnv(envar); !exists {
		defaultRegistry := &ecr.DescribeRegistryInput{}
		registry, err := fallback.DescribeRegistry(ctx, defaultRegistry)
		if err != nil {
			return "", err
		}
		id = *registry.RegistryId
	}

	return id, nil
}

func GetRegistryRegion(envar string, fallback aws.Config) string {
	var region string
	var exists bool

	if region, exists = os.LookupEnv(envar); !exists {
		region = fallback.Region
	}

	return region
}

func GetApiGatewayId(envar string, fallback ApiGatewayClient) (string, error) {
	var apis *apigatewayv2.GetApisOutput
	var SelfDiscoveredIds []string
	var err error

	if givenId, exists := os.LookupEnv(envar); exists {
		return givenId, nil
	}

	apis, err = fallback.GetApis(context.Background(), &apigatewayv2.GetApisInput{})
	if err != nil {
		return "", err
	}

	for _, api := range apis.Items {
		if _, exists := api.Tags["SelfDiscovery"]; exists {
			SelfDiscoveredIds = append(SelfDiscoveredIds, *api.ApiId)
		}
	}

	if len(SelfDiscoveredIds) == 0 {
		log.Info().Msg("no API gateways found with SelfDiscovery tag")
		return "", nil
	}

	if len(SelfDiscoveredIds) > 1 {
		return "", fmt.Errorf("multiple API gateways found with SelfDiscovery tag, use %s to specify", envar)
	}

	log.Info().Str("api", SelfDiscoveredIds[0]).Msg("self-discovered API gateway")
	return SelfDiscoveredIds[0], nil
}
