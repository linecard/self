package mock

import (
	"context"
	"encoding/base64"
	"strings"
	"time"

	"github.com/linecard/self/pkg/convention/config"
	mockfixture "github.com/linecard/self/pkg/mock/fixture"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ecr"
	ecrtypes "github.com/aws/aws-sdk-go-v2/service/ecr/types"
	dockertypes "github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/stretchr/testify/mock"
)

type MockRegistryService struct {
	mock.Mock
}

func (m *MockRegistryService) InspectByDigest(ctx context.Context, registryId, repository, digest string) (dockertypes.ImageInspect, error) {
	args := m.Called(ctx, registryId, repository, digest)
	return args.Get(0).(dockertypes.ImageInspect), args.Error(1)
}

func (m *MockRegistryService) InspectByTag(ctx context.Context, registryId, repository, tag string) (dockertypes.ImageInspect, error) {
	args := m.Called(ctx, registryId, repository, tag)
	return args.Get(0).(dockertypes.ImageInspect), args.Error(1)
}

func (m *MockRegistryService) ImageUri(ctx context.Context, registryId, registryUrl, repository, tag string) (string, error) {
	args := m.Called(ctx, registryId, registryUrl, repository, tag)
	return args.String(0), args.Error(1)
}

func (m *MockRegistryService) List(ctx context.Context, registry, repository string) (ecr.DescribeImagesOutput, error) {
	args := m.Called(ctx, registry, repository)
	return args.Get(0).(ecr.DescribeImagesOutput), args.Error(1)
}

func (m *MockRegistryService) Delete(ctx context.Context, registryId, repository string, imageDigests []string) error {
	args := m.Called(ctx, registryId, repository, imageDigests)
	return args.Error(0)
}

func (m *MockRegistryService) Untag(ctx context.Context, registryId, repository, tag string) error {
	args := m.Called(ctx, registryId, repository, tag)
	return args.Error(0)
}

func MockImageInspect(config config.Config, created *time.Time) dockertypes.ImageInspect {
	digest := "sha256:mockDigest"

	if created == nil {
		now := time.Now()
		created = &now
	}

	return dockertypes.ImageInspect{
		ID: digest,
		RepoTags: []string{
			config.RepositoryUrl() + ":" + config.Git.Branch,
			config.RepositoryUrl() + ":" + config.Git.Sha,
		},
		RepoDigests: []string{
			config.RepositoryUrl() + "@" + digest,
		},
		Parent:          "sha256:mockParentDigest",
		Comment:         "Mocked release for testing purposes",
		Created:         created.String(),
		Container:       "",
		ContainerConfig: &container.Config{},
		DockerVersion:   "20.10.7",
		Author:          "",
		Config: &container.Config{
			Hostname:     "4d868704a896",
			Domainname:   "",
			User:         "",
			AttachStdin:  false,
			AttachStdout: false,
			AttachStderr: false,
			Tty:          false,
			OpenStdin:    false,
			StdinOnce:    false,
			Env:          nil,
			ArgsEscaped:  false,
			Image:        "",
			Volumes:      nil,
			OnBuild:      nil,
			WorkingDir:   "",
			Cmd:          nil,
			Entrypoint:   nil,
			Labels: map[string]string{
				"org.linecard.self.role":      mockfixture.Base64("role.json.tmpl"),
				"org.linecard.self.policy":    mockfixture.Base64("policy.json.tmpl"),
				"org.linecard.self.bus":       mockfixture.Base64("bus.json.tmpl"),
				"org.linecard.self.resources": mockfixture.Base64("resources.json.tmpl"),
				"org.linecard.self.git-sha":   Base64(config.Git.Sha),
			},
		},
		Architecture: "arm64",
		Variant:      "",
		Os:           "linux",
		OsVersion:    "",
		Size:         0,
		VirtualSize:  0,
		GraphDriver:  dockertypes.GraphDriverData{},
		RootFS: dockertypes.RootFS{
			Type:   "",
			Layers: nil,
		},
		Metadata: image.Metadata{},
	}
}

func MockDescribeImagesOutput() ecr.DescribeImagesOutput {
	return ecr.DescribeImagesOutput{
		ImageDetails: []ecrtypes.ImageDetail{
			{
				ImageDigest: aws.String("sha256:mockDigestOne"),
				ImageTags: []string{
					"feature-branch",
					"1f3509a373489706fdf88d67b115905bffe92e1b",
				},
				ImageSizeInBytes: aws.Int64(0),
				ImagePushedAt:    aws.Time(time.Now()),
			},
			{
				ImageDigest: aws.String("sha256:mockDigestTwo"),
				ImageTags: []string{
					"alternate-branch",
					"03a4e4574a80273760663456e0a6bef6945d6abd",
				},
				ImageSizeInBytes: aws.Int64(0),
				ImagePushedAt:    aws.Time(time.Now()),
			},
		},
	}
}

func Base64(s string) string {
	var builder strings.Builder
	encoder := base64.NewEncoder(base64.StdEncoding, &builder)
	encoder.Write([]byte(s))
	encoder.Close()

	return builder.String()
}
