package credentials

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExtractAuthForHost(t *testing.T) {
	hostname := "registry.test"
	input := []byte(`{"auths":{"registry.test":{"username":"steve-o","password":"awesome"}}}`)

	t.Run("success", func(t *testing.T) {
		authConfigs, err := ExtractAuthConfigs(input)
		require.NoError(t, err)

		username, password, err := ExtractBasicAuthForHost(authConfigs, hostname)
		require.NoError(t, err)

		assert.Equal(t, "steve-o", username)
		assert.Equal(t, "awesome", password)
	})

	t.Run("bad_host", func(t *testing.T) {
		authConfigs, err := ExtractAuthConfigs(input)
		require.NoError(t, err)

		_, _, err = ExtractBasicAuthForHost(authConfigs, "other-host.com")
		assert.Error(t, err)
	})

	t.Run("bad_input", func(t *testing.T) {
		_, err := ExtractAuthConfigs([]byte("poo"))
		assert.Error(t, err)
	})
}
