package deployment

import (
	"context"
	"fmt"
	"testing"

	"github.com/linecard/self/pkg/convention/config"
	"github.com/linecard/self/pkg/convention/release"
	repomock "github.com/linecard/self/pkg/mock/repo"
	servicemock "github.com/linecard/self/pkg/mock/service"
	umweltmock "github.com/linecard/self/pkg/mock/umwelt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	"github.com/aws/aws-sdk-go-v2/service/lambda"
	"github.com/aws/aws-sdk-go-v2/service/lambda/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestDeployment(t *testing.T) {
	ctx := context.Background()

	awsConfig := aws.Config{Region: "us-west-2"}

	gitMock, cleanup := repomock.MockRepository("mockOrg", "mockRepo", "feature-branch", "function-one", "function-two", "function-three")
	defer cleanup()

	here := umweltmock.FromCwd(ctx, "mockRepo/function-one", gitMock, awsConfig)

	config := config.FromHere(here)

	getFunctionOutput := []lambda.GetFunctionOutput{
		*servicemock.MockGetFunctionOutput(config, "feature-branch", "function-one"),
		*servicemock.MockGetFunctionOutput(config, "feature-branch", "function-one"),
		*servicemock.MockGetFunctionOutput(config, "alternate-branch", "function-three"),
	}

	tests := []struct {
		name     string
		setup    func(*servicemock.MockFunctionService, *servicemock.MockRegistryService)
		teardown func(*servicemock.MockFunctionService, *servicemock.MockRegistryService)
		test     func(*testing.T, *servicemock.MockFunctionService, *servicemock.MockRegistryService)
	}{
		{
			name: "convention.Find calls service.Inspect correctly.",
			setup: func(mfs *servicemock.MockFunctionService, mrs *servicemock.MockRegistryService) {
				mfs.On("Inspect", mock.Anything, "mockRepo-feature-branch-function-one").Return(&getFunctionOutput[0], nil)
			},
			test: func(t *testing.T, mfs *servicemock.MockFunctionService, mrs *servicemock.MockRegistryService) {
				var expected Deployment

				deployments := FromServices(config, mfs, mrs)
				got, err := deployments.Find(ctx, "feature-branch", "function-one")
				expected = Deployment{getFunctionOutput[0]}

				assert.NoError(t, err)
				assert.EqualValuesf(t, expected, got, "expected %v, got %v", expected, got)
				assert.IsType(t, Deployment{}, got)
			},
		},
		{
			name: "convention.Find returns error when service.Inspect returns an error.",
			setup: func(mfs *servicemock.MockFunctionService, mrs *servicemock.MockRegistryService) {
				mfs.On("Inspect", mock.Anything, "mockRepo-feature-branch-function-2000").Return((*lambda.GetFunctionOutput)(nil), fmt.Errorf("not found"))
			},
			test: func(t *testing.T, mfs *servicemock.MockFunctionService, mrs *servicemock.MockRegistryService) {
				deployments := FromServices(config, mfs, mrs)
				got, err := deployments.Find(ctx, "feature-branch", "function-2000")

				assert.Error(t, err)
				assert.IsType(t, Deployment{}, got)
			},
		},
		{
			name: "convention.List calls service.List correctly.",
			setup: func(mfs *servicemock.MockFunctionService, mrs *servicemock.MockRegistryService) {
				mfs.On("List", mock.Anything, "mockRepo").Return(getFunctionOutput, nil)
			},
			test: func(t *testing.T, mfs *servicemock.MockFunctionService, mrs *servicemock.MockRegistryService) {
				var expected []Deployment

				deployments := FromServices(config, mfs, mrs)

				got, err := deployments.List(ctx)
				for _, output := range getFunctionOutput {
					expected = append(expected, Deployment{output})
				}

				assert.NoError(t, err)
				assert.EqualValuesf(t, expected, got, "expected %v, got %v", expected, got)
				assert.IsType(t, []Deployment{}, got)
			},
		},
		{
			name: "convention.List returns error when service.List returns an error.",
			setup: func(mfs *servicemock.MockFunctionService, mrs *servicemock.MockRegistryService) {
				mfs.On("List", mock.Anything, "mockRepo").Return(([]lambda.GetFunctionOutput)(nil), fmt.Errorf("not found"))
			},
			test: func(t *testing.T, mfs *servicemock.MockFunctionService, mrs *servicemock.MockRegistryService) {
				deployments := FromServices(config, mfs, mrs)
				list, err := deployments.List(ctx)
				assert.Error(t, err)
				assert.IsType(t, []Deployment{}, list)
			},
		},
		{
			name: "convention.ListNameSpace calls service.List and filters namespace tags correctly.",
			setup: func(mfs *servicemock.MockFunctionService, mrs *servicemock.MockRegistryService) {
				mfs.On("List", mock.Anything, "mockRepo").Return(getFunctionOutput, nil)
			},
			test: func(t *testing.T, mfs *servicemock.MockFunctionService, mrs *servicemock.MockRegistryService) {
				deployments := FromServices(config, mfs, mrs)
				list, err := deployments.ListNameSpace(ctx, "alternate-branch")
				assert.NoError(t, err)
				assert.Len(t, list, 1)
				assert.EqualValues(t, Deployment{getFunctionOutput[2]}, list[0])
				assert.IsType(t, []Deployment{}, list)
			},
		},
		{
			name: "connvention.Deploy calls service.PutFunction correctly.",
			setup: func(mfs *servicemock.MockFunctionService, mrs *servicemock.MockRegistryService) {

				// Mocking of the registry service here is a bit of crossing streams, but better than duplicating the logic... I think.
				mockInspect := servicemock.MockImageInspect(config, nil)
				mrs.On("InspectByTag", mock.Anything, "123456789013", "mockOrg/mockRepo/function-one", "feature-branch").Return(mockInspect, nil)
				mrs.On("ImageUri", mock.Anything, config.Registry.Id, config.Registry.Url, "mockOrg/mockRepo/function-one", "feature-branch").Return("123456789013.dkr.ecr.us-west-2.amazonaws.com/mockOrg/mockRepo/function-one@sha256:mockDigest", nil)

				tags := map[string]string{
					"NameSpace": "feature-branch",
					"Function":  "function-one",
					"Sha":       config.Git.Sha,
				}

				getRoleOutput := servicemock.MockGetRoleOutput(config, "feature-branch", "function-one")
				mfs.On("PutRole", mock.Anything, "mockRepo-feature-branch-function-one", mock.Anything, tags).Return(getRoleOutput, nil)

				getPolicyOutput := servicemock.MockGetPolicyOutput(config, "feature-branch", "function-one")
				mfs.On("PutPolicy", mock.Anything, "arn:aws:iam::123456789012:policy/mockRepo-feature-branch-function-one", mock.Anything, tags).Return(getPolicyOutput, nil)

				attachOutput := &iam.AttachRolePolicyOutput{}
				mfs.On("AttachPolicyToRole", mock.Anything, *getPolicyOutput.Policy.Arn, *getRoleOutput.Role.RoleName).Return(attachOutput, nil)

				mfs.On("PutFunction", mock.Anything,
					"mockRepo-feature-branch-function-one",
					"arn:aws:iam::123456789012:role/mockRepo-feature-branch-function-one",
					"123456789013.dkr.ecr.us-west-2.amazonaws.com/mockOrg/mockRepo/function-one@sha256:mockDigest",
					types.ArchitectureArm64,
					int32(1024),
					int32(128),
					int32(60),
					mock.MatchedBy(func(subnetIds []string) bool { return len(subnetIds) == 0 }),
					mock.MatchedBy(func(subnetIds []string) bool { return len(subnetIds) == 0 }),
					tags).Return(&getFunctionOutput[0], nil)

				mfs.On("Inspect", mock.Anything, "mockRepo-feature-branch-function-one").Return(&getFunctionOutput[0], nil)
			},
			test: func(t *testing.T, mfs *servicemock.MockFunctionService, mrs *servicemock.MockRegistryService) {
				var expected Deployment

				releases := release.FromServices(config, mrs, &servicemock.MockBuildService{})
				deployments := FromServices(config, mfs, mrs)

				mockRelease, err := releases.Find(ctx, "feature-branch")
				assert.NoError(t, err)

				got, err := deployments.Deploy(ctx, mockRelease, "feature-branch", "function-one")
				expected = Deployment{getFunctionOutput[0]}

				assert.NoError(t, err)
				assert.EqualValues(t, expected, got)
				assert.IsType(t, Deployment{}, got)
			},
		},
		// {
		// 	name: "convention.Destroy calls service.DeleteFunction correctly.",
		// 	setup: func(mfs *servicemock.MockFunctionService, mrs *servicemock.MockRegistryService) {
		// 		mfs.On("GetRolePolicies", mock.Anything, "mockRepo-feature-branch-function-one").Return(&iam.ListAttachedRolePoliciesOutput{
		// 			AttachedPolicies: []iamtypes.AttachedPolicy{
		// 				{
		// 					PolicyArn:  aws.String("arn:aws:iam::123456789012:policy/mockRepo-feature-branch-function-one"),
		// 					PolicyName: aws.String("mockRepo-feature-branch-function-one"),
		// 				},
		// 				{
		// 					PolicyArn:  aws.String("arn:aws:iam::123456789012:policy/mockRepo-feature-branch-function-one-unknown"),
		// 					PolicyName: aws.String("mockRepo-feature-branch-function-one-unknown"),
		// 				},
		// 			},
		// 		}, nil)

		// 		mfs.On("DetachPolicyFromRole", mock.Anything, "arn:aws:iam::123456789012:policy/mockRepo-feature-branch-function-one", "mockRepo-feature-branch-function-one").Return(&iam.DetachRolePolicyOutput{}, nil)
		// 		mfs.On("DetachPolicyFromRole", mock.Anything, "arn:aws:iam::123456789012:policy/mockRepo-feature-branch-function-one-unknown", "mockRepo-feature-branch-function-one").Return(&iam.DetachRolePolicyOutput{}, nil)
		// 		mfs.On("DeletePolicy", mock.Anything, "arn:aws:iam::123456789012:policy/mockRepo-feature-branch-function-one").Return(&iam.DeletePolicyOutput{}, nil)
		// 		mfs.On("DeletePolicy", mock.Anything, "arn:aws:iam::123456789012:policy/mockRepo-feature-branch-function-one-unknown").Return(&iam.DeletePolicyOutput{}, nil)
		// 		mfs.On("DeleteRole", mock.Anything, "mockRepo-feature-branch-function-one").Return(&iam.DeleteRoleOutput{}, nil)
		// 		mfs.On("DeleteFunction", mock.Anything, "mockRepo-feature-branch-function-one").Return(&lambda.DeleteFunctionOutput{}, nil)
		// 	},
		// 	test: func(t *testing.T, mfs *servicemock.MockFunctionService, mrs *servicemock.MockRegistryService) {
		// 		deployments := FromServices(config, mfs, mrs)
		// 		deployment := Deployment{getFunctionOutput[0]}
		// 		err := deployments.Destroy(ctx, deployment)
		// 		assert.NoError(t, err)
		// 	},
		// },
		// {
		// 	name: "convention.FetchRelease calls service.InspectByDigest correctly.",
		// 	setup: func(mfs *servicemock.MockFunctionService, mrs *servicemock.MockRegistryService) {
		// 		mockInspect := servicemock.MockImageInspect(config, nil)
		// 		mrs.On("InspectByDigest", ctx, "789012345678", "mockOrg/mockRepo/function-one", "sha256:mockDigest").Return(mockInspect, nil)
		// 	},
		// 	test: func(t *testing.T, mfs *servicemock.MockFunctionService, mrs *servicemock.MockRegistryService) {
		// 		deployment := Deployment{getFunctionOutput[0]}
		// 		fetched, err := deployment.FetchRelease(ctx, mrs, "789012345678")
		// 		assert.NoError(t, err)
		// 		assert.IsType(t, release.Release{}, fetched)
		// 	},
		// },
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mfs := &servicemock.MockFunctionService{}
			mrs := &servicemock.MockRegistryService{}

			if tc.setup != nil {
				tc.setup(mfs, mrs)
			}

			tc.test(t, mfs, mrs)

			if tc.teardown != nil {
				tc.teardown(mfs, mrs)
			}
		})
	}
}
