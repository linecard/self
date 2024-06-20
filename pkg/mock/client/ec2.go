package mocks

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/ec2"

	"github.com/stretchr/testify/mock"
)

type MockEc2Client struct {
	mock.Mock
}

func (m *MockEc2Client) DescribeVpcs(ctx context.Context, params *ec2.DescribeVpcsInput, optFns ...func(*ec2.Options)) (*ec2.DescribeVpcsOutput, error) {
	args := m.Called(ctx, params)
	return args.Get(0).(*ec2.DescribeVpcsOutput), args.Error(1)
}

func (m *MockEc2Client) DescribeSubnets(ctx context.Context, params *ec2.DescribeSubnetsInput, optFns ...func(*ec2.Options)) (*ec2.DescribeSubnetsOutput, error) {
	args := m.Called(ctx, params)
	return args.Get(0).(*ec2.DescribeSubnetsOutput), args.Error(1)
}
