package types

import "github.com/docker/docker/api/types"

// AuthConfigs map a server url to a Docker authorization config.
type AuthConfigs map[string]types.AuthConfig
