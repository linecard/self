package umwelt

import (
	"testing"

	mockrepo "github.com/linecard/self/pkg/mock/repo"

	"github.com/stretchr/testify/assert"
)

// Tests for files.go
func TestSelfish(t *testing.T) {
	_, cleanup := mockrepo.MockRepository("mockOrg", "mockRepo", "feature-branch", "function-1")
	defer cleanup()

	result := Selfish("mockRepo/function-1")
	assert.NotNil(t, result)
	assert.Equalf(t, "function-1", result.Name, "expected %s, got %s", "function-1", result.Name)
	assert.Equalf(t, "mockRepo/function-1", result.Path, "expected %s, got %s", "testpath", result.Path)
}

func TestSelfDiscovery(t *testing.T) {
	_, cleanup := mockrepo.MockRepository("mockOrg", "mockRepo", "feature-branch", "function-1", "function-2")
	defer cleanup()

	result := SelfDiscovery("mockRepo")
	assert.NotNil(t, result)
	assert.Len(t, result, 2)
	assert.Equal(t, result[0].Name, "function-1")
	assert.Equal(t, result[0].Path, "mockRepo/function-1")
	assert.Equal(t, result[1].Name, "function-2")
	assert.Equal(t, result[1].Path, "mockRepo/function-2")
}
