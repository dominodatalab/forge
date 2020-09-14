package credentials

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExtractDockerAuth(t *testing.T) {
	hostname := "registry.test"
	input := []byte(`{"auths":{"registry.test":{"username":"steve-o","password":"awesome"}}}`)

	t.Run("success", func(t *testing.T) {
		username, password, err := ExtractDockerAuth(input, hostname)
		require.NoError(t, err)
		assert.Equal(t, "steve-o", username)
		assert.Equal(t, "awesome", password)
	})

	t.Run("bad_host", func(t *testing.T) {
		_, _, err := ExtractDockerAuth(input, "other-host.com")
		assert.Error(t, err)
	})

	t.Run("bad_input", func(t *testing.T) {
		_, _, err := ExtractDockerAuth([]byte("poo"), hostname)
		assert.Error(t, err)
	})
}
