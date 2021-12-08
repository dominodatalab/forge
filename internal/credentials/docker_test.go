package credentials

import (
	"testing"

	"github.com/docker/docker/api/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExtractAuthConfigs(t *testing.T) {
	hostname := "registry.test"
	username := "steve-o"
	password := "awesome"

	input := []byte(`{"auths":{"registry.test":{"username":"steve-o","password":"awesome"}}}`)

	t.Run("success", func(t *testing.T) {
		authConfigs, err := ExtractAuthConfigs(input)
		require.NoError(t, err)
		require.Len(t, authConfigs, 1)
		require.Contains(t, authConfigs, hostname)

		authConfig := authConfigs[hostname]
		assert.Equal(t, username, authConfig.Username)
		assert.Equal(t, password, authConfig.Password)
		assert.Equal(t, hostname, authConfig.ServerAddress)
	})

	t.Run("bad_input", func(t *testing.T) {
		_, err := ExtractAuthConfigs([]byte("poo"))
		assert.Error(t, err)
	})
}

func TestExtractAuthForHost(t *testing.T) {
	hostname := "registry.test"
	username := "steve-o"
	password := "awesome"

	authConfigs := AuthConfigs{hostname: types.AuthConfig{
		ServerAddress: hostname,
		Username:      username,
		Password:      password,
	}}

	t.Run("success", func(t *testing.T) {
		username, password, err := ExtractBasicAuthForHost(authConfigs, hostname)
		require.NoError(t, err)

		assert.Equal(t, username, username)
		assert.Equal(t, password, password)
	})

	t.Run("bad_host", func(t *testing.T) {
		_, _, err := ExtractBasicAuthForHost(authConfigs, "other-host.com")
		assert.Error(t, err)
	})
}
