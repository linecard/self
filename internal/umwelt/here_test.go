package umwelt

import (
	"context"
	"net/url"
	"os"
	"testing"

	clientmock "github.com/linecard/self/pkg/mock/client"
	mockrepo "github.com/linecard/self/pkg/mock/repo"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ecr"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"github.com/linecard/self/internal/gitlib"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func defaultSetup(ctx context.Context, mockECR *clientmock.MockECRClient, mockSTS *clientmock.MockSTSClient) {
	mockSTS.On("GetCallerIdentity", ctx, mock.Anything).Return(&sts.GetCallerIdentityOutput{
		UserId:  aws.String("user-123"),
		Account: aws.String("123456789012"),
		Arn:     aws.String("arn:aws:iam::123456789012:user/test"),
	}, nil)

	mockECR.On("DescribeRegistry", ctx, mock.Anything).Return(&ecr.DescribeRegistryOutput{
		RegistryId: aws.String("123456789013"),
	}, nil)
}

func TestFromCwd(t *testing.T) {
	ctx := context.Background()
	awsConfig := aws.Config{Region: "us-west-2"}

	mockGit, cleanup := mockrepo.MockRepository("mockOrg", "mockRepo", "feature-branch", "function-one", "function-two")
	defer cleanup()

	cases := []struct {
		name     string
		setup    func(*clientmock.MockECRClient, *clientmock.MockSTSClient)
		teardown func(*clientmock.MockECRClient, *clientmock.MockSTSClient)
		test     func(*testing.T, *clientmock.MockECRClient, *clientmock.MockSTSClient)
	}{
		{
			name: "Repo Folder Scope: function nil, functions populated",
			setup: func(mecr *clientmock.MockECRClient, msts *clientmock.MockSTSClient) {
				os.Setenv("AWS_API_GATEWAY_ID", "ApiGatewayIdFromEnv")
				os.Setenv("AWS_SECURITY_GROUP_IDS", "sg-123,sg-456")
				os.Setenv("AWS_SUBNET_IDS", "sn-123,sn-456")
				defaultSetup(ctx, mecr, msts)
			},
			test: func(t *testing.T, mecr *clientmock.MockECRClient, msts *clientmock.MockSTSClient) {
				here, err := FromCwd(ctx, "mockRepo", mockGit, awsConfig, mecr, msts)

				ought := defaultFromCwd(mockGit, "mockRepo")
				ought.ApiGateway.Id = aws.String("ApiGatewayIdFromEnv")
				ought.Vpc.SecurityGroupIds = []string{"sg-123", "sg-456"}
				ought.Vpc.SubnetIds = []string{"sn-123", "sn-456"}

				assert.NoError(t, err)
				assert.EqualValues(t, ought, here)
			},
			teardown: func(mecr *clientmock.MockECRClient, msts *clientmock.MockSTSClient) {
				os.Unsetenv("AWS_API_GATEWAY_ID")
				os.Unsetenv("AWS_SECURITY_GROUP_IDS")
				os.Unsetenv("AWS_SUBNET_IDS")
			},
		},
		{
			name: "Function Folder Scope: function populated, functions populated",
			setup: func(mecr *clientmock.MockECRClient, msts *clientmock.MockSTSClient) {
				defaultSetup(ctx, mecr, msts)
			},
			test: func(t *testing.T, mecr *clientmock.MockECRClient, msts *clientmock.MockSTSClient) {
				here, err := FromCwd(ctx, "mockRepo/function-one", mockGit, awsConfig, mecr, msts)

				ought := defaultFromCwd(mockGit, "mockRepo/function-one")
				ought.Function = &ThisFunction{
					Name: "function-one",
					Path: mockGit.Root + "/function-one",
				}

				assert.NoError(t, err)
				assert.EqualValues(t, ought, here)
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			mecr := &clientmock.MockECRClient{}
			msts := &clientmock.MockSTSClient{}

			if tc.setup != nil {
				tc.setup(mecr, msts)
			}

			tc.test(t, mecr, msts)

			if tc.teardown != nil {
				tc.teardown(mecr, msts)
			}
		})
	}
}

func defaultFromCwd(mockGit gitlib.DotGit, cwd string) Here {
	return Here{
		Cwd: cwd,
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
			Id: nil,
		},
		Vpc: ThisVpc{
			SecurityGroupIds: nil,
			SubnetIds:        nil,
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

func TestFromEvent(t *testing.T) {
	ctx := context.Background()
	awsConfig := aws.Config{Region: "us-west-2"}

	cases := []struct {
		name     string
		setup    func(*clientmock.MockECRClient, *clientmock.MockSTSClient)
		teardown func(*clientmock.MockECRClient, *clientmock.MockSTSClient)
		test     func(*testing.T, *clientmock.MockECRClient, *clientmock.MockSTSClient)
	}{
		{
			name: "Event with Branch Tag",
			setup: func(mecr *clientmock.MockECRClient, msts *clientmock.MockSTSClient) {
				os.Setenv("AWS_API_GATEWAY_ID", "ApiGatewayIdFromEnv")
				os.Setenv("AWS_SUBNET_IDS", "sn-123,sn-456")
				os.Setenv("AWS_SECURITY_GROUP_IDS", "sg-123,sg-456")
				defaultSetup(ctx, mecr, msts)
			},
			test: func(t *testing.T, mecr *clientmock.MockECRClient, msts *clientmock.MockSTSClient) {
				event := events.ECRImageActionEvent{
					DetailType: "ECR Image Action",
					Detail: events.ECRImageActionEventDetailType{
						ActionType:     "PUSH",
						RepositoryName: "organization/repo/function_one",
						ImageTag:       "branchName",
					},
				}

				ought := defaultFromEvent()
				ought.Git.Branch = "branchName"
				ought.ApiGateway.Id = aws.String("ApiGatewayIdFromEnv")
				ought.Vpc.SecurityGroupIds = []string{"sg-123", "sg-456"}
				ought.Vpc.SubnetIds = []string{"sn-123", "sn-456"}

				here, err := FromEvent(ctx, event, awsConfig, mecr, msts)
				assert.NoError(t, err)
				assert.EqualValues(t, ought, here)
			},
			teardown: func(*clientmock.MockECRClient, *clientmock.MockSTSClient) {
				os.Unsetenv("AWS_API_GATEWAY_ID")
				os.Unsetenv("AWS_SUBNET_IDS")
				os.Unsetenv("AWS_SECURITY_GROUP_IDS")
			},
		},
		{
			name: "Event with SHA tag",
			setup: func(mecr *clientmock.MockECRClient, msts *clientmock.MockSTSClient) {
				defaultSetup(ctx, mecr, msts)
			},
			test: func(t *testing.T, mecr *clientmock.MockECRClient, msts *clientmock.MockSTSClient) {
				event := events.ECRImageActionEvent{
					DetailType: "ECR Image Action",
					Detail: events.ECRImageActionEventDetailType{
						ActionType:     "DELETE",
						RepositoryName: "organization/repo/function_one",
						ImageTag:       "2e17ab2c190fc5dfff79e66fc972f015da937f05",
					},
				}

				ought := defaultFromEvent()
				ought.Git.Sha = "2e17ab2c190fc5dfff79e66fc972f015da937f05"

				here, err := FromEvent(ctx, event, awsConfig, mecr, msts)
				assert.NoError(t, err)
				assert.EqualValues(t, ought, here)
			},
		},
		{
			name: "Event with no env config",
			setup: func(mecr *clientmock.MockECRClient, msts *clientmock.MockSTSClient) {
				defaultSetup(ctx, mecr, msts)
			},
			test: func(t *testing.T, mecr *clientmock.MockECRClient, msts *clientmock.MockSTSClient) {
				event := events.ECRImageActionEvent{
					DetailType: "ECR Image Action",
					Detail: events.ECRImageActionEventDetailType{
						ActionType:     "DELETE",
						RepositoryName: "organization/repo/function_one",
						ImageTag:       "branchName",
					},
				}

				ought := defaultFromEvent()
				ought.Git.Branch = "branchName"

				here, err := FromEvent(ctx, event, awsConfig, mecr, msts)
				assert.NoError(t, err)
				assert.EqualValues(t, ought, here)
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			mecr := &clientmock.MockECRClient{}
			msts := &clientmock.MockSTSClient{}

			if tc.setup != nil {
				tc.setup(mecr, msts)
			}

			tc.test(t, mecr, msts)

			if tc.teardown != nil {
				tc.teardown(mecr, msts)
			}
		})
	}
}

func defaultFromEvent() Here {
	return Here{
		Cwd: "",
		Caller: ThisCaller{
			Id:      "user-123",
			Account: "123456789012",
			Arn:     "arn:aws:iam::123456789012:user/test",
			Region:  "us-west-2",
		},
		Git: gitlib.DotGit{
			Origin: &url.URL{
				Scheme: "https",
				Host:   "github.com",
				Path:   "/organization/repo",
			},
		},
		ApiGateway: ThisApiGateway{
			Id: nil,
		},
		Vpc: ThisVpc{
			SecurityGroupIds: nil,
			SubnetIds:        nil,
		},
		Registry: ThisRegistry{
			Id:     "123456789013",
			Region: "us-west-2",
		},
		Function: &ThisFunction{
			Name: "function_one",
		},
	}
}
