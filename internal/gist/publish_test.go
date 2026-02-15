package gist

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPublish_Success(t *testing.T) {
	mockRun := func(name string, args ...string) ([]byte, error) {
		assert.Equal(t, "gh", name)
		assert.Contains(t, args, "gist")
		assert.Contains(t, args, "create")
		assert.Contains(t, args, "/tmp/test.html")
		return []byte("https://gist.github.com/abc123\n"), nil
	}

	url, err := publish("/tmp/test.html", "", false, mockRun)
	require.NoError(t, err)
	assert.Equal(t, "https://gist.github.com/abc123", url)
}

func TestPublish_WithDescription(t *testing.T) {
	mockRun := func(name string, args ...string) ([]byte, error) {
		assert.Contains(t, args, "--desc")
		assert.Contains(t, args, "My session log")
		return []byte("https://gist.github.com/abc123\n"), nil
	}

	url, err := publish("/tmp/test.html", "My session log", false, mockRun)
	require.NoError(t, err)
	assert.Equal(t, "https://gist.github.com/abc123", url)
}

func TestPublish_Public(t *testing.T) {
	mockRun := func(name string, args ...string) ([]byte, error) {
		assert.Contains(t, args, "--public")
		return []byte("https://gist.github.com/abc123\n"), nil
	}

	url, err := publish("/tmp/test.html", "", true, mockRun)
	require.NoError(t, err)
	assert.Equal(t, "https://gist.github.com/abc123", url)
}

func TestPublish_Error(t *testing.T) {
	mockRun := func(name string, args ...string) ([]byte, error) {
		return []byte("not authenticated"), fmt.Errorf("exit status 1")
	}

	_, err := publish("/tmp/test.html", "", false, mockRun)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "gh gist create failed")
	assert.Contains(t, err.Error(), "not authenticated")
}

func TestPublish_NotPublicByDefault(t *testing.T) {
	mockRun := func(name string, args ...string) ([]byte, error) {
		for _, a := range args {
			assert.NotEqual(t, "--public", a)
		}
		return []byte("https://gist.github.com/abc123\n"), nil
	}

	_, err := publish("/tmp/test.html", "", false, mockRun)
	require.NoError(t, err)
}
