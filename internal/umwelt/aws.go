package umwelt

import (
	"context"
	"fmt"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/apigatewayv2"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
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

func GetApiGatewayId(envar string, discovery ApiGatewayClient) (string, error) {
	var apiId string
	var exists bool
	var err error

	if apiId, exists = os.LookupEnv(envar); !exists {
		return "", nil
	}

	getApisOutput, err := discovery.GetApis(context.Background(), &apigatewayv2.GetApisInput{})
	if err != nil {
		return "", err
	}

	for _, api := range getApisOutput.Items {
		if _, exists := api.Tags["SelfDiscovery"]; exists {
			if *api.ApiId == apiId {
				return apiId, nil
			}
		}
	}

	return "", fmt.Errorf("no api found with id %s and tagged with SelfDiscovery", apiId)
}

func GetVpcIds(envar string, discovery Ec2Client) (string, []string, error) {
	var vpcId string
	var subnetIds []string
	var exists bool

	if vpcId, exists = os.LookupEnv(envar); !exists {
		return "", []string{}, nil
	}

	describeVpcsOutput, err := discovery.DescribeVpcs(context.Background(), &ec2.DescribeVpcsInput{
		Filters: []types.Filter{
			{
				Name:   aws.String("vpc-id"),
				Values: []string{vpcId},
			},
			{
				Name:   aws.String("tag-key"),
				Values: []string{"SelfDiscovery"},
			},
		},
	})

	if err != nil {
		return "", []string{}, err
	}

	log.Info().Msgf("discovered %d vpcs", len(describeVpcsOutput.Vpcs))

	if len(describeVpcsOutput.Vpcs) == 0 {
		return "", []string{}, fmt.Errorf("no VPC found with id %s and tagged with SelfDiscovery", vpcId)
	}

	if len(describeVpcsOutput.Vpcs) > 1 {
		return "", []string{}, fmt.Errorf("multiple VPCs found with id %s and tagged with SelfDiscovery", vpcId)
	}

	describeSubnetsOutput, err := discovery.DescribeSubnets(context.Background(), &ec2.DescribeSubnetsInput{
		Filters: []types.Filter{
			{
				Name:   aws.String("vpc-id"),
				Values: []string{vpcId},
			},
			{
				Name:   aws.String("tag-key"),
				Values: []string{"SelfDiscovery"},
			},
		},
	})

	if err != nil {
		return "", []string{}, err
	}

	log.Info().Msgf("discovered %d subnets", len(describeSubnetsOutput.Subnets))

	if len(describeSubnetsOutput.Subnets) == 0 {
		return "", []string{}, fmt.Errorf("VPC %s contains no subnets tagged with SelfDiscovery", vpcId)
	}

	for _, subnet := range describeSubnetsOutput.Subnets {
		for _, tag := range subnet.Tags {
			if *tag.Key == "SelfDiscovery" {
				subnetIds = append(subnetIds, *subnet.SubnetId)
			}
		}
	}

	return vpcId, subnetIds, nil
}
