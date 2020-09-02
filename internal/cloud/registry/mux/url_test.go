package mux

import (
	"context"
	"regexp"
	"testing"

	dockertypes "github.com/docker/docker/api/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dominodatalab/forge/internal/cloud/registry/types"
)

func TestURLMux(t *testing.T) {
	expected := types.AuthConfigs{
		"test-server": dockertypes.AuthConfig{},
	}
	mux := NewURLMux()
	mux.RegisterLoader(regexp.MustCompile(`^my.cloud`), func(ctx context.Context, url string) (types.AuthConfigs, error) {
		return expected, nil
	})

	t.Run("loader_match", func(t *testing.T) {
		loader, err := mux.FromString("my.cloud/best/cloud")
		require.NoError(t, err)

		out, _ := loader(context.TODO(), "")
		assert.Equal(t, expected, out)
	})

	t.Run("loader_missing", func(t *testing.T) {
		_, err := mux.FromString("your.cloud/silly/cloud")
		assert.IsType(t, NoLoaderFoundError{}, err)
	})
}
