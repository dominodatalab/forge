package credentials

import "github.com/docker/docker/api/types"

// AuthConfigs is a map of registry urls to authentication credentials.
type AuthConfigs map[string]types.AuthConfig

// DockerConfigJSON models the structure of .dockerconfigfile data.
type DockerConfigJSON struct {
	Auths AuthConfigs `json:"auths"`
}
