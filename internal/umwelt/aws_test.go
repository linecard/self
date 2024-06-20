package umwelt

import (
	"context"
	"os"
	"testing"

	clientmock "github.com/linecard/self/pkg/mock/client"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/apigatewayv2"
	"github.com/aws/aws-sdk-go-v2/service/apigatewayv2/types"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/aws/aws-sdk-go-v2/service/ecr"
	"github.com/stretchr/testify/assert"
)

func TestAWSPerception(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name     string
		setup    func(*clientmock.MockECRClient, *clientmock.MockApiGatewayClient, *clientmock.MockEc2Client)
		teardown func(*clientmock.MockECRClient, *clientmock.MockApiGatewayClient, *clientmock.MockEc2Client)
		test     func(*testing.T, *clientmock.MockECRClient, *clientmock.MockApiGatewayClient, *clientmock.MockEc2Client)
	}{
		{
			name: "ECR discovery: from account",
			setup: func(mecr *clientmock.MockECRClient, mgw *clientmock.MockApiGatewayClient, mec2 *clientmock.MockEc2Client) {
				os.Unsetenv("AWS_REGISTRY_ID")
				os.Unsetenv("AWS_REGISTRY_REGION")

				mecr.On("DescribeRegistry", ctx, &ecr.DescribeRegistryInput{}).Return(&ecr.DescribeRegistryOutput{
					RegistryId: aws.String("fetched_default_registry_id_from_account"),
				}, nil)
			},
			test: func(t *testing.T, mecr *clientmock.MockECRClient, mgw *clientmock.MockApiGatewayClient, mec2 *clientmock.MockEc2Client) {
				registryID, err := GetRegistryId(ctx, "AWS_REGISTRY_ID", mecr)
				assert.NoError(t, err)
				assert.Equal(t, "fetched_default_registry_id_from_account", registryID)
			},
		},
		{
			name: "ECR discovery: from env",
			setup: func(mecr *clientmock.MockECRClient, mgw *clientmock.MockApiGatewayClient, mec2 *clientmock.MockEc2Client) {
				os.Setenv("AWS_REGISTRY_ID", "env_registry_id")
				os.Setenv("AWS_REGISTRY_REGION", "env_registry_region")
			},
			test: func(t *testing.T, mecr *clientmock.MockECRClient, mgw *clientmock.MockApiGatewayClient, mec2 *clientmock.MockEc2Client) {
				registryID, err := GetRegistryId(ctx, "AWS_REGISTRY_ID", mecr)
				assert.NoError(t, err)
				assert.Equal(t, "env_registry_id", registryID)
			},
			teardown: func(mecr *clientmock.MockECRClient, mgw *clientmock.MockApiGatewayClient, mec2 *clientmock.MockEc2Client) {
				os.Unsetenv("AWS_REGISTRY_ID")
				os.Unsetenv("AWS_REGISTRY_REGION")
			},
		},
		{
			name: "API Gateway discovery: env set and two tagged gateways available",
			setup: func(mecr *clientmock.MockECRClient, mgw *clientmock.MockApiGatewayClient, mec2 *clientmock.MockEc2Client) {
				os.Setenv("AWS_API_GATEWAY_ID", "ApiGatewayIdFromEnv")

				mgw.On("GetApis", ctx, &apigatewayv2.GetApisInput{}).Return(&apigatewayv2.GetApisOutput{
					Items: []types.Api{
						{
							ApiId: aws.String("CorrectlyConfiguredButWrongId"),
							Tags:  map[string]string{"SelfDiscovery": "sense"},
						},
						{
							ApiId: aws.String("ApiGatewayIdFromEnv"),
							Tags:  map[string]string{"SelfDiscovery": "sense"},
						},
					},
				}, nil)
			},
			test: func(t *testing.T, mecr *clientmock.MockECRClient, mgw *clientmock.MockApiGatewayClient, mec2 *clientmock.MockEc2Client) {
				apiID, err := GetApiGatewayId("AWS_API_GATEWAY_ID", mgw)
				assert.NoError(t, err)
				assert.Equal(t, "ApiGatewayIdFromEnv", apiID)
			},
			teardown: func(mecr *clientmock.MockECRClient, mgw *clientmock.MockApiGatewayClient, mec2 *clientmock.MockEc2Client) {
				os.Unsetenv("AWS_API_GATEWAY_ID")
			},
		},
		{
			name: "API Gateway discovery: env selects untagged gateway",
			setup: func(mecr *clientmock.MockECRClient, mgw *clientmock.MockApiGatewayClient, mec2 *clientmock.MockEc2Client) {
				os.Setenv("AWS_API_GATEWAY_ID", "ApiGatewayIdFromEnv")

				mgw.On("GetApis", ctx, &apigatewayv2.GetApisInput{}).Return(&apigatewayv2.GetApisOutput{
					Items: []types.Api{
						{
							ApiId: aws.String("CorrectlyConfiguredButWrongId"),
							Tags:  map[string]string{"SelfDiscovery": "sense"},
						},
						{
							ApiId: aws.String("ApiGatewayIdFromEnv"),
						},
					},
				}, nil)
			},
			test: func(t *testing.T, mecr *clientmock.MockECRClient, mgw *clientmock.MockApiGatewayClient, mec2 *clientmock.MockEc2Client) {
				apiID, err := GetApiGatewayId("AWS_API_GATEWAY_ID", mgw)
				assert.Error(t, err)
				assert.Equal(t, "", apiID)
			},
			teardown: func(mecr *clientmock.MockECRClient, mgw *clientmock.MockApiGatewayClient, mec2 *clientmock.MockEc2Client) {
				os.Unsetenv("AWS_API_GATEWAY_ID")
			},
		},
		{
			name: "API Gateway discovery: no env set",
			setup: func(mecr *clientmock.MockECRClient, mgw *clientmock.MockApiGatewayClient, mec2 *clientmock.MockEc2Client) {
				os.Unsetenv("AWS_API_GATEWAY_ID")

				mgw.On("GetApis", ctx, &apigatewayv2.GetApisInput{}).Return(&apigatewayv2.GetApisOutput{
					Items: []types.Api{
						{
							ApiId: aws.String("CorrectlyConfiguredButWrongId"),
							Tags:  map[string]string{"SelfDiscovery": "sense"},
						},
						{
							ApiId: aws.String("ApiGatewayIdFromEnv"),
							Tags:  map[string]string{"SelfDiscovery": "sense"},
						},
					},
				}, nil)
			},
			test: func(t *testing.T, mecr *clientmock.MockECRClient, mgw *clientmock.MockApiGatewayClient, mec2 *clientmock.MockEc2Client) {
				apiID, err := GetApiGatewayId("AWS_API_GATEWAY_ID", mgw)
				assert.NoError(t, err)
				assert.Equal(t, "", apiID)
			},
		},
		{
			name: "VPC and Subnet: env set and vpc plus subnets tagged",
			setup: func(mecr *clientmock.MockECRClient, mgw *clientmock.MockApiGatewayClient, mec2 *clientmock.MockEc2Client) {
				os.Setenv("AWS_VPC_ID", "VpcIdFromEnv")

				mec2.On("DescribeVpcs", ctx, &ec2.DescribeVpcsInput{
					Filters: []ec2types.Filter{
						{
							Name:   aws.String("vpc-id"),
							Values: []string{"VpcIdFromEnv"},
						},
						{
							Name:   aws.String("tag-key"),
							Values: []string{"SelfDiscovery"},
						},
					},
				}).Return(&ec2.DescribeVpcsOutput{
					Vpcs: []ec2types.Vpc{
						{
							VpcId: aws.String("VpcIdFromEnv"),
						},
					}}, nil)

				mec2.On("DescribeSubnets", ctx, &ec2.DescribeSubnetsInput{
					Filters: []ec2types.Filter{
						{
							Name:   aws.String("vpc-id"),
							Values: []string{"VpcIdFromEnv"},
						},
						{
							Name:   aws.String("tag-key"),
							Values: []string{"SelfDiscovery"},
						},
					},
				}).Return(&ec2.DescribeSubnetsOutput{
					Subnets: []ec2types.Subnet{
						{
							SubnetId: aws.String("SubnetIdFromEnv_1"),
							Tags:     []ec2types.Tag{{Key: aws.String("SelfDiscovery"), Value: aws.String("sense")}},
						},
						{
							SubnetId: aws.String("SubnetIdFromEnv_2"),
							Tags:     []ec2types.Tag{{Key: aws.String("SelfDiscovery"), Value: aws.String("sense")}},
						},
						{
							SubnetId: aws.String("SubnetIdFromEnv_3"),
						},
					},
				}, nil)
			},
			test: func(t *testing.T, mecr *clientmock.MockECRClient, mgw *clientmock.MockApiGatewayClient, mec2 *clientmock.MockEc2Client) {
				vpcID, subnetIDs, err := GetVpcIds("AWS_VPC_ID", mec2)
				assert.NoError(t, err)
				assert.Equal(t, "VpcIdFromEnv", vpcID)
				assert.Equal(t, len(subnetIDs), 2)
			},
			teardown: func(mecr *clientmock.MockECRClient, mgw *clientmock.MockApiGatewayClient, mec2 *clientmock.MockEc2Client) {
				os.Unsetenv("AWS_VPC_ID")
			},
		},
		{
			name: "VPC and Subnet: env set and vpc configured but no tagged subnets",
			setup: func(mecr *clientmock.MockECRClient, mgw *clientmock.MockApiGatewayClient, mec2 *clientmock.MockEc2Client) {
				os.Setenv("AWS_VPC_ID", "VpcIdFromEnv")

				mec2.On("DescribeVpcs", ctx, &ec2.DescribeVpcsInput{
					Filters: []ec2types.Filter{
						{
							Name:   aws.String("vpc-id"),
							Values: []string{"VpcIdFromEnv"},
						},
						{
							Name:   aws.String("tag-key"),
							Values: []string{"SelfDiscovery"},
						},
					},
				}).Return(&ec2.DescribeVpcsOutput{
					Vpcs: []ec2types.Vpc{
						{
							VpcId: aws.String("VpcIdFromEnv"),
						},
					}}, nil)

				mec2.On("DescribeSubnets", ctx, &ec2.DescribeSubnetsInput{
					Filters: []ec2types.Filter{
						{
							Name:   aws.String("vpc-id"),
							Values: []string{"VpcIdFromEnv"},
						},
						{
							Name:   aws.String("tag-key"),
							Values: []string{"SelfDiscovery"},
						},
					},
				}).Return(&ec2.DescribeSubnetsOutput{
					Subnets: []ec2types.Subnet{},
				}, nil)
			},
			test: func(t *testing.T, mecr *clientmock.MockECRClient, mgw *clientmock.MockApiGatewayClient, mec2 *clientmock.MockEc2Client) {
				vpcID, subnetIDs, err := GetVpcIds("AWS_VPC_ID", mec2)
				assert.Error(t, err)
				assert.Equal(t, "", vpcID)
				assert.Equal(t, len(subnetIDs), 0)
			},
			teardown: func(mecr *clientmock.MockECRClient, mgw *clientmock.MockApiGatewayClient, mec2 *clientmock.MockEc2Client) {
				os.Unsetenv("AWS_VPC_ID")
			},
		},
		{
			name: "VPC and Subnet: env not set",
			setup: func(mecr *clientmock.MockECRClient, mgw *clientmock.MockApiGatewayClient, mec2 *clientmock.MockEc2Client) {
				os.Unsetenv("AWS_VPC_ID")
			},
			test: func(t *testing.T, mecr *clientmock.MockECRClient, mgw *clientmock.MockApiGatewayClient, mec2 *clientmock.MockEc2Client) {
				vpcID, subnetIDs, err := GetVpcIds("AWS_VPC_ID", mec2)
				assert.NoError(t, err)
				assert.Equal(t, "", vpcID)
				assert.Equal(t, len(subnetIDs), 0)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mecr := &clientmock.MockECRClient{}
			mgw := &clientmock.MockApiGatewayClient{}
			mec2 := &clientmock.MockEc2Client{}

			if tc.setup != nil {
				tc.setup(mecr, mgw, mec2)
			}

			tc.test(t, mecr, mgw, mec2)

			if tc.teardown != nil {
				tc.teardown(mecr, mgw, mec2)
			}
		})
	}
}
