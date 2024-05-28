package umwelt

import (
	"context"
	"fmt"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/apigatewayv2"
	"github.com/aws/aws-sdk-go-v2/service/ecr"
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
		return "", fmt.Errorf("no API gateways found with tag %s", "SelfDiscovery")
	}

	if len(SelfDiscoveredIds) > 1 {
		return "", fmt.Errorf("multiple self-discovered APIs found, must declare intent via %s", envar)
	}

	return SelfDiscoveredIds[0], nil
}
