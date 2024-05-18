package release

import (
	"context"
	"encoding/base64"
	"testing"

	"github.com/linecard/self/convention/config"
	mockfixture "github.com/linecard/self/mock/fixture"
	mockrepo "github.com/linecard/self/mock/repo"
	mockservice "github.com/linecard/self/mock/service"
	mockumwelt "github.com/linecard/self/mock/umwelt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestRelease(t *testing.T) {
	ctx := context.Background()

	awsConfig := aws.Config{Region: "us-west-2"}

	gitMock, cleanup := mockrepo.MockRepository("mockOrg", "mockRepo", "feature-branch", "function-one", "function-two")
	defer cleanup()

	here := mockumwelt.FromCwd(ctx, "mockRepo/function-one", gitMock, awsConfig)
	config := config.FromHere(here)

	cases := []struct {
		name     string
		setup    func(mbs *mockservice.MockBuildService, mrs *mockservice.MockRegistryService)
		teardown func(mbs *mockservice.MockBuildService, mrs *mockservice.MockRegistryService)
		test     func(t *testing.T, mbs *mockservice.MockBuildService, mrs *mockservice.MockRegistryService)
	}{
		{
			name: "convention.Find calls service.InspectByTag correctly",
			setup: func(mbs *mockservice.MockBuildService, mrs *mockservice.MockRegistryService) {
				mockInspect := mockservice.MockImageInspect(config, nil)
				mrs.On("InspectByTag", mock.Anything, config.Registry.Id, config.Repository.Prefix+"/"+config.Function.Name, "feature-branch").Return(mockInspect, nil)
				mrs.On("ImageUri", mock.Anything, config.Registry.Id, config.Registry.Url, config.Repository.Prefix+"/"+config.Function.Name, "feature-branch").Return("mockUri", nil)
			},
			test: func(t *testing.T, mbs *mockservice.MockBuildService, mrs *mockservice.MockRegistryService) {
				releases := FromServices(config, mrs, mbs)
				release, err := releases.Find(ctx, "feature-branch")
				assert.NoError(t, err)
				assert.Equal(t, release.Uri, "mockUri")
			},
		},
		{
			name: "convention.List calls service.List correctly",
			setup: func(mbs *mockservice.MockBuildService, mrs *mockservice.MockRegistryService) {
				mockImages := mockservice.MockDescribeImagesOutput()
				mrs.On("List", mock.Anything, config.Registry.Url, config.Repository.Prefix+"/"+config.Function.Name).Return(mockImages, nil)
			},
			test: func(t *testing.T, mbs *mockservice.MockBuildService, mrs *mockservice.MockRegistryService) {
				releases := FromServices(config, mrs, mbs)
				images, err := releases.List(ctx, config.Function.Name)
				assert.NoError(t, err)
				assert.Equal(t, len(images), 2)
				assert.IsType(t, []ReleaseSummary{}, images)
			},
		},
		{
			name: "convention.Build calls service.Build correctly",
			setup: func(mbs *mockservice.MockBuildService, mrs *mockservice.MockRegistryService) {
				expectedLabels := map[string]string{
					"org.linecard.self.bus.default.bus": mockfixture.Base64("bus.json.tmpl"),
					"org.linecard.self.policy":          mockfixture.Base64("policy.json.tmpl"),
					"org.linecard.self.role":            mockfixture.Base64("role.json.tmpl"),
					"org.linecard.self.git-sha":         base64.StdEncoding.EncodeToString([]byte(config.Git.Sha)),
				}

				expectedTags := []string{
					"123456789013.dkr.ecr.us-west-2.amazonaws.com/mockOrg/mockRepo/function-one:feature-branch",
					"123456789013.dkr.ecr.us-west-2.amazonaws.com/mockOrg/mockRepo/function-one:4eeac06a6b0d37a30bd45775b19e59a06c3b6295",
				}

				mbs.On("Build", mock.Anything, config.Function.Path, expectedLabels, expectedTags).Return((error)(nil))

				mockImageInspect := mockservice.MockImageInspect(config, nil)
				mbs.On("InspectByTag", mock.Anything, config.Registry.Url, config.Repository.Prefix+"/"+config.Function.Name, "4eeac06a6b0d37a30bd45775b19e59a06c3b6295").Return(mockImageInspect, (error)(nil))
			},
			test: func(t *testing.T, mbs *mockservice.MockBuildService, mrs *mockservice.MockRegistryService) {
				releases := FromServices(config, mrs, mbs)
				image, err := releases.Build(ctx, config.Function.Path, config.Git.Branch, config.Git.Sha)
				assert.IsType(t, Image{}, image)
				assert.NoError(t, err)
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			mbs := &mockservice.MockBuildService{}
			mrs := &mockservice.MockRegistryService{}

			if tc.setup != nil {
				tc.setup(mbs, mrs)
			}

			tc.test(t, mbs, mrs)

			if tc.teardown != nil {
				tc.teardown(mbs, mrs)
			}
		})
	}
}
