package umwelt

import (
	"context"
	"os"
	"testing"

	clientmock "github.com/linecard/self/mock/client"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ecr"
	"github.com/stretchr/testify/assert"
)

func TestDeploymentFind(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name     string
		setup    func(*clientmock.MockECRClient, *clientmock.MockApiGatewayClient)
		teardown func(*clientmock.MockECRClient, *clientmock.MockApiGatewayClient)
		test     func(*testing.T, *clientmock.MockECRClient, *clientmock.MockApiGatewayClient)
	}{
		{
			name: "ECR discovery: from account",
			setup: func(mecr *clientmock.MockECRClient, mgw *clientmock.MockApiGatewayClient) {
				os.Unsetenv("AWS_REGISTRY_ID")
				os.Unsetenv("AWS_REGISTRY_REGION")

				mecr.On("DescribeRegistry", ctx, &ecr.DescribeRegistryInput{}).Return(&ecr.DescribeRegistryOutput{
					RegistryId: aws.String("fetched_default_registry_id_from_account"),
				}, nil)
			},
			test: func(t *testing.T, mecr *clientmock.MockECRClient, mgw *clientmock.MockApiGatewayClient) {
				registryID, err := GetRegistryId(ctx, "AWS_REGISTRY_ID", mecr)
				assert.NoError(t, err)
				assert.Equal(t, "fetched_default_registry_id_from_account", registryID)
			},
		},
		{
			name: "ECR discovery: from env",
			setup: func(mecr *clientmock.MockECRClient, mgw *clientmock.MockApiGatewayClient) {
				os.Setenv("AWS_REGISTRY_ID", "env_registry_id")
				os.Setenv("AWS_REGISTRY_REGION", "env_registry_region")
			},
			test: func(t *testing.T, mecr *clientmock.MockECRClient, mgw *clientmock.MockApiGatewayClient) {
				registryID, err := GetRegistryId(ctx, "AWS_REGISTRY_ID", mecr)
				assert.NoError(t, err)
				assert.Equal(t, "env_registry_id", registryID)
			},
			teardown: func(mecr *clientmock.MockECRClient, mgw *clientmock.MockApiGatewayClient) {
				os.Unsetenv("AWS_REGISTRY_ID")
				os.Unsetenv("AWS_REGISTRY_REGION")
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mecr := &clientmock.MockECRClient{}
			mgw := &clientmock.MockApiGatewayClient{}

			if tc.setup != nil {
				tc.setup(mecr, mgw)
			}

			tc.test(t, mecr, mgw)

			if tc.teardown != nil {
				tc.teardown(mecr, mgw)
			}
		})
	}
}
