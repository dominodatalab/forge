package types

import (
	"context"

	"github.com/docker/docker/api/types"
)

// AuthConfigs map a server url to a Docker authorization config.
type AuthConfigs map[string]types.AuthConfig

// AuthLoader is function that must be implemented by every cloud-specific registry authorization provider.
type AuthLoader func(ctx context.Context, url string) (AuthConfigs, error)
