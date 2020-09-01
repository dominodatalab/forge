package mux

import (
	"context"
	"errors"
	"regexp"
	"testing"

	dockertypes "github.com/docker/docker/api/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dominodatalab/forge/internal/cloud/registry/types"
)

func TestURLMux(t *testing.T) {
	var expected types.AuthConfigs

	ctx := context.Background()
	mux := NewURLMux()
	mux.RegisterLoader(regexp.MustCompile(`^my.cloud`), func(ctx context.Context, url string) (types.AuthConfigs, error) {
		expected = types.AuthConfigs{
			url: dockertypes.AuthConfig{Username: "steve", Password: "o"},
		}
		return expected, nil
	})
	mux.RegisterLoader(regexp.MustCompile(`^bad.cloud`), func(ctx context.Context, url string) (types.AuthConfigs, error) {
		return nil, errors.New("arbitrary azure limit")
	})

	t.Run("loader_match", func(t *testing.T) {
		url := "my.cloud/best/cloud"
		loader, err := mux.FromString(url)
		require.NoError(t, err)

		actual, err := loader(ctx, url)
		require.NoError(t, err)
		assert.Equal(t, expected, actual)
	})

	t.Run("loader_missing", func(t *testing.T) {
		_, err := mux.FromString("your.cloud/silly/cloud")
		assert.EqualError(t, err, `no loader found for "your.cloud/silly/cloud"`)
	})

	t.Run("loader_err", func(t *testing.T) {
		url := "bad.cloud/sad/cloud"
		loader, err := mux.FromString(url)
		require.NoError(t, err)

		_, err = loader(ctx, url)
		assert.EqualError(t, err, "arbitrary azure limit")
	})
}
