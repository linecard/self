package config

import (
	"context"
	"testing"

	"github.com/linecard/self/internal/gitlib"
	repomock "github.com/linecard/self/pkg/mock/repo"
	umweltmock "github.com/linecard/self/pkg/mock/umwelt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/stretchr/testify/assert"
)

func TestFromHere(t *testing.T) {
	ctx := context.Background()

	awsConfig := aws.Config{Region: "us-west-2"}

	mockGit, cleanup := repomock.MockRepository("mockOrg", "mockRepo", "feature-branch", "function-one", "function-two")
	defer cleanup()

	here := umweltmock.FromCwd(ctx, "mockRepo/function-one", mockGit, awsConfig)

	expected := defaultExpectation(mockGit)
	got := FromHere(here)

	// test struct equality
	assert.EqualValuesf(t, got, expected, "%v failed", "Produces correct config from given here")

	// test computed values
	expectedResourceName := "mockRepo-feature-branch-function-one"
	gotResourceName := got.ResourceName(got.Git.Branch, got.Function.Name)
	assert.EqualValuesf(t, expectedResourceName, gotResourceName, "%v failed", "computes correct resource name")

	expectedRepositoryName := "mockOrg/mockRepo/function-one"
	gotResourcePath := got.RepositoryName()
	assert.EqualValuesf(t, expectedRepositoryName, gotResourcePath, "%v failed", "computes correct repository name")

	expectedRepositoryUrl := "123456789013.dkr.ecr.us-west-2.amazonaws.com/mockOrg/mockRepo/function-one"
	gotRepositoryUrl := got.RepositoryUrl()
	assert.EqualValuesf(t, expectedRepositoryUrl, gotRepositoryUrl, "%v failed", "computes correct repository url")

	expectedRouteKey := "ANY /mockRepo/feature-branch/function-one/{proxy+}"
	gotRouteKey := got.RouteKey(got.Git.Branch)
	assert.EqualValuesf(t, expectedRouteKey, gotRouteKey, "%v failed", "computes correct route key")
}

func defaultExpectation(mockGit gitlib.DotGit) Config {
	return Config{
		Function: &Function{
			Name: "function-one",
			Path: "mockRepo/function-one",
		},
		Functions: []Function{
			{
				Name: "function-one",
				Path: "mockRepo/function-one",
			},
			{
				Name: "function-two",
				Path: "mockRepo/function-two",
			},
		},
		Caller: Caller{
			Arn: "arn:aws:iam::123456789012:user/test",
		},
		Account: Account{
			Id:     "123456789012",
			Region: "us-west-2",
		},
		Registry: Registry{
			Id:     "123456789013",
			Region: "us-west-2",
			Url:    "123456789013.dkr.ecr.us-west-2.amazonaws.com",
		},
		Vpc: Vpc{
			SubnetIds:        nil,
			SecurityGroupIds: nil,
		},
		ApiGateway: ApiGateway{
			Id: nil,
		},
		Git: Git{
			Origin: "https://github.com/mockOrg/mockRepo.git",
			Branch: mockGit.Branch,
			Sha:    mockGit.Sha,
			Root:   mockGit.Root,
			Dirty:  false,
		},
		Repository: Repository{
			Prefix: "mockOrg/mockRepo",
		},
		Resource: Resource{
			Prefix: "mockRepo",
		},
		TemplateData: TemplateData{
			AccountId:         "123456789012",
			Region:            "us-west-2",
			RegistryRegion:    "us-west-2",
			RegistryAccountId: "123456789013",
		},
		Labels: Labels{
			Schema: StringLabel{
				Description: "Label schema version string",
				Key:         "org.linecard.self.schema",
				Content:     "1.0",
			},
			Sha: StringLabel{
				Description: "Git sha string",
				Key:         "org.linecard.self.git-sha",
				Content:     mockGit.Sha,
			},
			Role: EmbeddedFileLabel{
				Description: "Role template file",
				Key:         "org.linecard.self.role",
				Path:        "embedded/roles/lambda.json.tmpl",
				Required:    true,
			},
			Policy: FileLabel{
				Description: "Policy template file",
				Key:         "org.linecard.self.policy",
				Path:        "mockRepo/function-one/policy.json.tmpl",
				Required:    true,
			},
			Resources: FileLabel{
				Description: "Resources template file",
				Key:         "org.linecard.self.resources",
				Path:        "mockRepo/function-one/resources.json.tmpl",
			},
			Bus: FolderLabel{
				Description: "Bus templates folder",
				KeyPrefix:   "org.linecard.self.bus",
				Path:        "mockRepo/function-one/bus",
			},
		},
	}
}
