package mock

import (
	"context"

	"github.com/docker/docker/api/types"
	"github.com/stretchr/testify/mock"
)

// MockBuildService is a mock of BuildService interface
type MockBuildService struct {
	mock.Mock
}

func (m *MockBuildService) InspectByTag(ctx context.Context, registryUrl, repository, tag string) (types.ImageInspect, error) {
	args := m.Called(ctx, registryUrl, repository, tag)
	return args.Get(0).(types.ImageInspect), args.Error(1)
}

func (m *MockBuildService) Build(ctx context.Context, path string, labels map[string]string, tags []string) error {
	args := m.Called(ctx, path, labels, tags)
	return args.Error(0)
}

func (m *MockBuildService) Push(ctx context.Context, tag string) error {
	args := m.Called(ctx, tag)
	return args.Error(0)
}
