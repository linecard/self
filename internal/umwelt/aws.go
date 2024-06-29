package umwelt

import (
	"context"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
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
