package mocks

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/apigatewayv2"
	"github.com/stretchr/testify/mock"
)

type MockApiGatewayClient struct {
	mock.Mock
}

func (m *MockApiGatewayClient) GetApis(ctx context.Context, params *apigatewayv2.GetApisInput, optFns ...func(*apigatewayv2.Options)) (*apigatewayv2.GetApisOutput, error) {
	args := m.Called(ctx, params)
	return args.Get(0).(*apigatewayv2.GetApisOutput), args.Error(1)
}
