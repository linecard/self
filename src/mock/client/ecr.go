package mocks

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/ecr"
	"github.com/stretchr/testify/mock"
)

type MockECRClient struct {
	mock.Mock
}

func (m *MockECRClient) DescribeRegistry(ctx context.Context, params *ecr.DescribeRegistryInput, optFns ...func(*ecr.Options)) (*ecr.DescribeRegistryOutput, error) {
	args := m.Called(ctx, params)
	return args.Get(0).(*ecr.DescribeRegistryOutput), args.Error(1)
}
