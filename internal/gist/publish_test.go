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
		require.Len(t, args, 3)
		assert.Equal(t, "gist", args[0])
		assert.Equal(t, "create", args[1])
		assert.Equal(t, "/tmp/test.html", args[2])
		return []byte("https://gist.github.com/abc123\n"), nil
	}

	url, err := publish("/tmp/test.html", "", false, mockRun)
	require.NoError(t, err)
	assert.Equal(t, "https://gist.github.com/abc123", url)
}

func TestPublish_WithDescription(t *testing.T) {
	mockRun := func(name string, args ...string) ([]byte, error) {
		// args: gist create /tmp/test.html --desc "My session log"
		require.Len(t, args, 5)
		assert.Equal(t, "gist", args[0])
		assert.Equal(t, "create", args[1])
		assert.Equal(t, "/tmp/test.html", args[2])
		assert.Equal(t, "--desc", args[3])
		assert.Equal(t, "My session log", args[4])
		return []byte("https://gist.github.com/abc123\n"), nil
	}

	url, err := publish("/tmp/test.html", "My session log", false, mockRun)
	require.NoError(t, err)
	assert.Equal(t, "https://gist.github.com/abc123", url)
}

func TestPublish_Public(t *testing.T) {
	mockRun := func(name string, args ...string) ([]byte, error) {
		// args: gist create /tmp/test.html --public
		require.Len(t, args, 4)
		assert.Equal(t, "--public", args[3])
		return []byte("https://gist.github.com/abc123\n"), nil
	}

	url, err := publish("/tmp/test.html", "", true, mockRun)
	require.NoError(t, err)
	assert.Equal(t, "https://gist.github.com/abc123", url)
}

func TestPublish_PublicWithDescription(t *testing.T) {
	mockRun := func(name string, args ...string) ([]byte, error) {
		// args: gist create /tmp/test.html --desc "desc" --public
		require.Len(t, args, 6)
		assert.Equal(t, "--desc", args[3])
		assert.Equal(t, "desc", args[4])
		assert.Equal(t, "--public", args[5])
		return []byte("https://gist.github.com/abc123\n"), nil
	}

	url, err := publish("/tmp/test.html", "desc", true, mockRun)
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
		// Without --public, should be exactly 3 args
		require.Len(t, args, 3, "should not include --public when public=false")
		return []byte("https://gist.github.com/abc123\n"), nil
	}

	_, err := publish("/tmp/test.html", "", false, mockRun)
	require.NoError(t, err)
}

func TestPublish_TrimsWhitespace(t *testing.T) {
	mockRun := func(name string, args ...string) ([]byte, error) {
		return []byte("  https://gist.github.com/abc123  \n"), nil
	}

	url, err := publish("/tmp/test.html", "", false, mockRun)
	require.NoError(t, err)
	assert.Equal(t, "https://gist.github.com/abc123", url, "should trim whitespace from URL")
}
