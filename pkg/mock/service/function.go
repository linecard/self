package mock

import (
	"context"
	"fmt"
	"time"

	"github.com/linecard/self/pkg/convention/config"
	fixturemock "github.com/linecard/self/pkg/mock/fixture"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	iamtypes "github.com/aws/aws-sdk-go-v2/service/iam/types"
	"github.com/aws/aws-sdk-go-v2/service/lambda"
	"github.com/aws/aws-sdk-go-v2/service/lambda/types"
	"github.com/stretchr/testify/mock"
)

// MockFunctionService is a mock of FunctionService interface
type MockFunctionService struct {
	mock.Mock
}

func (m *MockFunctionService) Inspect(ctx context.Context, name string) (*lambda.GetFunctionOutput, error) {
	args := m.Called(ctx, name)
	return args.Get(0).(*lambda.GetFunctionOutput), args.Error(1)
}

func (m *MockFunctionService) List(ctx context.Context, prefix string) ([]lambda.GetFunctionOutput, error) {
	args := m.Called(ctx, prefix)
	return args.Get(0).([]lambda.GetFunctionOutput), args.Error(1)
}

func (m *MockFunctionService) PutPolicy(ctx context.Context, arn string, document string, tags map[string]string) (*iam.GetPolicyOutput, error) {
	args := m.Called(ctx, arn, document, tags)
	return args.Get(0).(*iam.GetPolicyOutput), args.Error(1)
}

func (m *MockFunctionService) DeletePolicy(ctx context.Context, arn string) (*iam.DeletePolicyOutput, error) {
	args := m.Called(ctx, arn)
	return args.Get(0).(*iam.DeletePolicyOutput), args.Error(1)
}

func (m *MockFunctionService) PutRole(ctx context.Context, name string, document string, tags map[string]string) (*iam.GetRoleOutput, error) {
	args := m.Called(ctx, name, document, tags)
	return args.Get(0).(*iam.GetRoleOutput), args.Error(1)
}

func (m *MockFunctionService) DeleteRole(ctx context.Context, name string) (*iam.DeleteRoleOutput, error) {
	args := m.Called(ctx, name)
	return args.Get(0).(*iam.DeleteRoleOutput), args.Error(1)
}

func (m *MockFunctionService) AttachPolicyToRole(ctx context.Context, policyArn, roleName string) (*iam.AttachRolePolicyOutput, error) {
	args := m.Called(ctx, policyArn, roleName)
	return args.Get(0).(*iam.AttachRolePolicyOutput), args.Error(1)
}

func (m *MockFunctionService) DetachPolicyFromRole(ctx context.Context, policyArn, roleName string) (*iam.DetachRolePolicyOutput, error) {
	args := m.Called(ctx, policyArn, roleName)
	return args.Get(0).(*iam.DetachRolePolicyOutput), args.Error(1)
}

func (m *MockFunctionService) PutFunction(ctx context.Context, name string, roleArn string, imageUri string, arch types.Architecture, ephemeralStorage, memorySize, timeout int32, subnetIds []string, tags map[string]string) (*lambda.GetFunctionOutput, error) {
	args := m.Called(ctx, name, roleArn, imageUri, arch, ephemeralStorage, memorySize, timeout, subnetIds, tags)
	return args.Get(0).(*lambda.GetFunctionOutput), args.Error(1)
}

func (m *MockFunctionService) DeleteFunction(ctx context.Context, name string) (*lambda.DeleteFunctionOutput, error) {
	args := m.Called(ctx, name)
	return args.Get(0).(*lambda.DeleteFunctionOutput), args.Error(1)
}

func (m *MockFunctionService) GetRolePolicies(ctx context.Context, name string) (*iam.ListAttachedRolePoliciesOutput, error) {
	args := m.Called(ctx, name)
	return args.Get(0).(*iam.ListAttachedRolePoliciesOutput), args.Error(1)
}

// Mock Responses

func MockGetFunctionOutput(config config.Config, namespace, functionName string) *lambda.GetFunctionOutput {
	resourceName := config.ResourceName(namespace, functionName)
	digest := "sha256:mockDigest"

	return &lambda.GetFunctionOutput{
		Code: &types.FunctionCodeLocation{
			ImageUri: aws.String(config.RepositoryUrl() + "@" + digest),
		},
		Configuration: &types.FunctionConfiguration{
			Architectures: []types.Architecture{
				types.ArchitectureArm64,
			},
			Role:        aws.String(fmt.Sprintf("arn:aws:iam::123456789012:role/%s", resourceName)),
			CodeSha256:  aws.String(digest),
			CodeSize:    1024,
			Description: aws.String("Mocked GetFunctionOutput for testing purposes"),
			EphemeralStorage: &types.EphemeralStorage{
				Size: aws.Int32(512),
			},
			Timeout:      aws.Int32(60),
			MemorySize:   aws.Int32(1024),
			FunctionArn:  aws.String(fmt.Sprintf("arn:aws:lambda:us-west-2:123456789012:function:%s", resourceName)),
			FunctionName: aws.String(resourceName),
			LastModified: aws.String("2021-07-01T00:00:00Z"),
		},
		Tags: map[string]string{
			"NameSpace": namespace,
			"Function":  functionName,
			"Sha":       config.Git.Sha,
		},
	}
}

func MockGetRoleOutput(config config.Config, namespace, functionName string) *iam.GetRoleOutput {
	resourceName := config.ResourceName(namespace, functionName)
	return &iam.GetRoleOutput{
		Role: &iamtypes.Role{
			Arn:                      aws.String("arn:aws:iam::123456789012:role/" + resourceName),
			RoleName:                 aws.String(resourceName),
			RoleId:                   aws.String("AIDAJQABLZS4A3QDU576Q"),
			CreateDate:               aws.Time(time.Now()),
			Path:                     aws.String("/"),
			AssumeRolePolicyDocument: aws.String(fixturemock.Read("role.json.tmpl")),
			PermissionsBoundary:      &iamtypes.AttachedPermissionsBoundary{},
		},
	}
}

func MockGetPolicyOutput(config config.Config, namespace, functionName string) *iam.GetPolicyOutput {
	resourceName := config.ResourceName(namespace, functionName)
	return &iam.GetPolicyOutput{
		Policy: &iamtypes.Policy{
			Arn:              aws.String("arn:aws:iam::123456789012:policy/" + resourceName),
			PolicyName:       aws.String(resourceName),
			PolicyId:         aws.String("AIDAJQABLZS4A3QDU576F"),
			CreateDate:       aws.Time(time.Now()),
			DefaultVersionId: aws.String("v1"),
			Path:             aws.String("/"),
			AttachmentCount:  aws.Int32(0),
		},
	}
}
