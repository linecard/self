package config

import (
	"context"
	"testing"

	"github.com/linecard/self/internal/gitlib"
	repomock "github.com/linecard/self/mock/repo"
	umweltmock "github.com/linecard/self/mock/umwelt"

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
		Label: Label{
			Role:      "org.linecard.self.role",
			Policy:    "org.linecard.self.policy",
			Sha:       "org.linecard.self.git-sha",
			Bus:       "org.linecard.self.bus",
			Resources: "org.linecard.self.resources",
		},
		Httproxy: Httproxy{
			ApiId: "mockApiId",
		},
	}
}
