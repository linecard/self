package umwelt

import (
	"context"
	"net/url"
	"testing"

	clientmock "github.com/linecard/self/mock/client"
	mockrepo "github.com/linecard/self/mock/repo"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/apigatewayv2"
	"github.com/aws/aws-sdk-go-v2/service/apigatewayv2/types"
	"github.com/aws/aws-sdk-go-v2/service/ecr"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"github.com/linecard/self/internal/gitlib"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func defaultSetup(ctx context.Context, mockSTS *clientmock.MockSTSClient, mockECR *clientmock.MockECRClient, mockApiGateway *clientmock.MockApiGatewayClient) {
	mockSTS.On("GetCallerIdentity", ctx, mock.Anything).Return(&sts.GetCallerIdentityOutput{
		UserId:  aws.String("user-123"),
		Account: aws.String("123456789012"),
		Arn:     aws.String("arn:aws:iam::123456789012:user/test"),
	}, nil)

	mockECR.On("DescribeRegistry", ctx, mock.Anything).Return(&ecr.DescribeRegistryOutput{
		RegistryId: aws.String("123456789013"),
	}, nil)

	mockApiGateway.On("GetApis", ctx, mock.Anything).Return(&apigatewayv2.GetApisOutput{
		Items: []types.Api{
			{
				ApiId: aws.String("mockApiId"),
				Tags:  map[string]string{"SelfManaged": "true"},
			},
		},
	}, nil)
}

func TestFromCwd(t *testing.T) {
	ctx := context.Background()
	awsConfig := aws.Config{Region: "us-west-2"}

	mockGit, cleanup := mockrepo.MockRepository("mockOrg", "mockRepo", "feature-branch", "function-one", "function-two")
	defer cleanup()

	cases := []struct {
		name     string
		setup    func(*clientmock.MockSTSClient, *clientmock.MockECRClient, *clientmock.MockApiGatewayClient)
		teardown func(*clientmock.MockSTSClient, *clientmock.MockECRClient, *clientmock.MockApiGatewayClient)
		test     func(*testing.T, *clientmock.MockSTSClient, *clientmock.MockECRClient, *clientmock.MockApiGatewayClient)
	}{
		{
			name: "Repo Folder Scope: function nil, functions populated",
			setup: func(msts *clientmock.MockSTSClient, mecr *clientmock.MockECRClient, mgw *clientmock.MockApiGatewayClient) {
				defaultSetup(ctx, msts, mecr, mgw)
			},
			test: func(t *testing.T, msts *clientmock.MockSTSClient, mecr *clientmock.MockECRClient, mgw *clientmock.MockApiGatewayClient) {
				here, err := FromCwd(ctx, "mockRepo", mockGit, awsConfig, mecr, mgw, msts)
				assert.NoError(t, err)
				assert.EqualValues(t, defaultHereExpectation(mockGit), here)
			},
		},
		{
			name: "Function Folder Scope: function populated, functions populated",
			setup: func(msts *clientmock.MockSTSClient, mecr *clientmock.MockECRClient, mgw *clientmock.MockApiGatewayClient) {
				defaultSetup(ctx, msts, mecr, mgw)
			},
			test: func(t *testing.T, msts *clientmock.MockSTSClient, mecr *clientmock.MockECRClient, mgw *clientmock.MockApiGatewayClient) {
				here, err := FromCwd(ctx, "mockRepo/function-one", mockGit, awsConfig, mecr, mgw, msts)
				assert.NoError(t, err)
				assert.EqualValues(t, functionScopeHereExpectation(defaultHereExpectation(mockGit)), here)
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			msts := &clientmock.MockSTSClient{}
			mecr := &clientmock.MockECRClient{}
			mgw := &clientmock.MockApiGatewayClient{}

			if tc.setup != nil {
				tc.setup(msts, mecr, mgw)
			}

			tc.test(t, msts, mecr, mgw)

			if tc.teardown != nil {
				tc.teardown(msts, mecr, mgw)
			}
		})
	}
}

func defaultHereExpectation(mockGit gitlib.DotGit) Here {
	return Here{
		Caller: ThisCaller{
			Id:      "user-123",
			Account: "123456789012",
			Arn:     "arn:aws:iam::123456789012:user/test",
			Region:  "us-west-2",
		},
		Git: mockGit,
		Registry: ThisRegistry{
			Id:     "123456789013",
			Region: "us-west-2",
		},
		ApiGateway: ThisApiGateway{
			Id: "mockApiId",
		},
		Function: nil,
		Functions: []ThisFunction{
			{
				Name: "function-one",
				Path: mockGit.Root + "/function-one",
			},
			{
				Name: "function-two",
				Path: mockGit.Root + "/function-two",
			},
		},
	}
}

func functionScopeHereExpectation(defaultHere Here) Here {
	return Here{
		Caller:     defaultHere.Caller,
		Git:        defaultHere.Git,
		Registry:   defaultHere.Registry,
		ApiGateway: defaultHere.ApiGateway,
		Function: &ThisFunction{
			Name: "function-one",
			Path: defaultHere.Git.Root + "/function-one",
		},
		Functions: defaultHere.Functions,
	}
}

func TestFromEvent(t *testing.T) {
	ctx := context.Background()
	awsConfig := aws.Config{Region: "us-west-2"}

	cases := []struct {
		name     string
		setup    func(*clientmock.MockSTSClient, *clientmock.MockECRClient, *clientmock.MockApiGatewayClient)
		teardown func(*clientmock.MockSTSClient, *clientmock.MockECRClient, *clientmock.MockApiGatewayClient)
		test     func(*testing.T, *clientmock.MockSTSClient, *clientmock.MockECRClient, *clientmock.MockApiGatewayClient)
	}{
		{
			name: "Event with Branch Tag",
			setup: func(msts *clientmock.MockSTSClient, mecr *clientmock.MockECRClient, mgw *clientmock.MockApiGatewayClient) {
				defaultSetup(ctx, msts, mecr, mgw)
			},
			test: func(t *testing.T, msts *clientmock.MockSTSClient, mecr *clientmock.MockECRClient, mgw *clientmock.MockApiGatewayClient) {
				event := events.ECRImageActionEvent{
					DetailType: "ECR Image Action",
					Detail: events.ECRImageActionEventDetailType{
						ActionType:     "PUSH",
						RepositoryName: "organization/repo/function_one",
						ImageTag:       "branchName",
					},
				}

				here, err := FromEvent(ctx, event, awsConfig, mecr, mgw, msts)
				assert.NoError(t, err)
				assert.EqualValues(t, branchEventExpectation(), here)
			},
		},
		{
			name: "Event with SHA tag",
			setup: func(msts *clientmock.MockSTSClient, mecr *clientmock.MockECRClient, mgw *clientmock.MockApiGatewayClient) {
				defaultSetup(ctx, msts, mecr, mgw)
			},
			test: func(t *testing.T, msts *clientmock.MockSTSClient, mecr *clientmock.MockECRClient, mgw *clientmock.MockApiGatewayClient) {
				event := events.ECRImageActionEvent{
					DetailType: "ECR Image Action",
					Detail: events.ECRImageActionEventDetailType{
						ActionType:     "DELETE",
						RepositoryName: "organization/repo/function_one",
						ImageTag:       "2e17ab2c190fc5dfff79e66fc972f015da937f05",
					},
				}

				here, err := FromEvent(ctx, event, awsConfig, mecr, mgw, msts)
				assert.NoError(t, err)
				assert.EqualValues(t, shaEventExpectation(branchEventExpectation()), here)
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			msts := &clientmock.MockSTSClient{}
			mecr := &clientmock.MockECRClient{}
			mgw := &clientmock.MockApiGatewayClient{}

			if tc.setup != nil {
				tc.setup(msts, mecr, mgw)
			}

			tc.test(t, msts, mecr, mgw)

			if tc.teardown != nil {
				tc.teardown(msts, mecr, mgw)
			}
		})
	}

}

func branchEventExpectation() Here {
	return Here{
		Caller: ThisCaller{
			Id:      "user-123",
			Account: "123456789012",
			Arn:     "arn:aws:iam::123456789012:user/test",
			Region:  "us-west-2",
		},
		Git: gitlib.DotGit{
			Branch: "branchName",
			Origin: &url.URL{
				Scheme: "https",
				Host:   "github.com",
				Path:   "/organization/repo",
			},
		},
		Registry: ThisRegistry{
			Id:     "123456789013",
			Region: "us-west-2",
		},
		ApiGateway: ThisApiGateway{
			Id: "mockApiId",
		},
		Function: &ThisFunction{
			Name: "function_one",
			Path: "",
		},
		Functions: nil,
	}
}

func shaEventExpectation(branchEventExpectation Here) Here {
	return Here{
		Caller: branchEventExpectation.Caller,
		Git: gitlib.DotGit{
			Branch: "",
			Sha:    "2e17ab2c190fc5dfff79e66fc972f015da937f05",
			Origin: branchEventExpectation.Git.Origin,
		},
		Registry:   branchEventExpectation.Registry,
		ApiGateway: branchEventExpectation.ApiGateway,
		Function: &ThisFunction{
			Name: "function_one",
			Path: "",
		},
		Functions: nil,
	}
}
