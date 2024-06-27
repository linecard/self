package function

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/iam"
	"github.com/aws/aws-sdk-go-v2/service/lambda"
)

type LambdaClient interface {
	GetFunction(ctx context.Context, params *lambda.GetFunctionInput, optFns ...func(*lambda.Options)) (*lambda.GetFunctionOutput, error)
	ListFunctions(ctx context.Context, params *lambda.ListFunctionsInput, optFns ...func(*lambda.Options)) (*lambda.ListFunctionsOutput, error)
	CreateFunction(ctx context.Context, params *lambda.CreateFunctionInput, optFns ...func(*lambda.Options)) (*lambda.CreateFunctionOutput, error)
	GetFunctionConfiguration(ctx context.Context, params *lambda.GetFunctionConfigurationInput, optFns ...func(*lambda.Options)) (*lambda.GetFunctionConfigurationOutput, error)
	UpdateFunctionConfiguration(ctx context.Context, params *lambda.UpdateFunctionConfigurationInput, optFns ...func(*lambda.Options)) (*lambda.UpdateFunctionConfigurationOutput, error)
	UpdateFunctionCode(ctx context.Context, params *lambda.UpdateFunctionCodeInput, optFns ...func(*lambda.Options)) (*lambda.UpdateFunctionCodeOutput, error)
	TagResource(ctx context.Context, params *lambda.TagResourceInput, optFns ...func(*lambda.Options)) (*lambda.TagResourceOutput, error)
	DeleteFunction(ctx context.Context, params *lambda.DeleteFunctionInput, optFns ...func(*lambda.Options)) (*lambda.DeleteFunctionOutput, error)
	PutFunctionConcurrency(ctx context.Context, params *lambda.PutFunctionConcurrencyInput, optFns ...func(*lambda.Options)) (*lambda.PutFunctionConcurrencyOutput, error)
}

type IamClient interface {
	GetPolicy(ctx context.Context, params *iam.GetPolicyInput, optFns ...func(*iam.Options)) (*iam.GetPolicyOutput, error)
	CreatePolicy(ctx context.Context, params *iam.CreatePolicyInput, optFns ...func(*iam.Options)) (*iam.CreatePolicyOutput, error)
	DeletePolicy(ctx context.Context, params *iam.DeletePolicyInput, optFns ...func(*iam.Options)) (*iam.DeletePolicyOutput, error)
	ListPolicyTags(context.Context, *iam.ListPolicyTagsInput, ...func(*iam.Options)) (*iam.ListPolicyTagsOutput, error)
	TagPolicy(ctx context.Context, params *iam.TagPolicyInput, optFns ...func(*iam.Options)) (*iam.TagPolicyOutput, error)
	UntagPolicy(ctx context.Context, params *iam.UntagPolicyInput, optFns ...func(*iam.Options)) (*iam.UntagPolicyOutput, error)
	ListPolicyVersions(ctx context.Context, params *iam.ListPolicyVersionsInput, optFns ...func(*iam.Options)) (*iam.ListPolicyVersionsOutput, error)
	CreatePolicyVersion(ctx context.Context, params *iam.CreatePolicyVersionInput, optFns ...func(*iam.Options)) (*iam.CreatePolicyVersionOutput, error)
	DeletePolicyVersion(ctx context.Context, params *iam.DeletePolicyVersionInput, optFns ...func(*iam.Options)) (*iam.DeletePolicyVersionOutput, error)
	ListPolicies(ctx context.Context, params *iam.ListPoliciesInput, optFns ...func(*iam.Options)) (*iam.ListPoliciesOutput, error)
	CreateRole(ctx context.Context, params *iam.CreateRoleInput, optFns ...func(*iam.Options)) (*iam.CreateRoleOutput, error)
	GetRole(ctx context.Context, params *iam.GetRoleInput, optFns ...func(*iam.Options)) (*iam.GetRoleOutput, error)
	DeleteRole(ctx context.Context, params *iam.DeleteRoleInput, optFns ...func(*iam.Options)) (*iam.DeleteRoleOutput, error)
	ListRoleTags(context.Context, *iam.ListRoleTagsInput, ...func(*iam.Options)) (*iam.ListRoleTagsOutput, error)
	UpdateAssumeRolePolicy(ctx context.Context, params *iam.UpdateAssumeRolePolicyInput, optFns ...func(*iam.Options)) (*iam.UpdateAssumeRolePolicyOutput, error)
	TagRole(ctx context.Context, params *iam.TagRoleInput, optFns ...func(*iam.Options)) (*iam.TagRoleOutput, error)
	UntagRole(ctx context.Context, params *iam.UntagRoleInput, optFns ...func(*iam.Options)) (*iam.UntagRoleOutput, error)
	ListAttachedRolePolicies(context.Context, *iam.ListAttachedRolePoliciesInput, ...func(*iam.Options)) (*iam.ListAttachedRolePoliciesOutput, error)
	AttachRolePolicy(ctx context.Context, params *iam.AttachRolePolicyInput, optFns ...func(*iam.Options)) (*iam.AttachRolePolicyOutput, error)
	DetachRolePolicy(ctx context.Context, params *iam.DetachRolePolicyInput, optFns ...func(*iam.Options)) (*iam.DetachRolePolicyOutput, error)
}

type Clients struct {
	Lambda LambdaClient
	Iam    IamClient
}

type Service struct {
	Client Clients
}

func FromClients(lambdaClient LambdaClient, iamClient IamClient) Service {
	return Service{
		Client: Clients{
			Lambda: lambdaClient,
			Iam:    iamClient,
		},
	}
}
