package credentials

import (
	"bytes"
	"fmt"
	"github.com/docker/cli/cli/config/configfile"
	"github.com/docker/docker/api/types"
)

// AuthConfigs is a map of registry urls to authentication credentials.
type AuthConfigs map[string]types.AuthConfig

// DockerConfigJSON models the structure of .dockerconfigfile data.
type DockerConfigJSON struct {
	Auths AuthConfigs `json:"auths"`
}

// ExtractAuthConfigs will return the Docker AuthConfigs from a JSON-representation of a Docker config file.
func ExtractAuthConfigs(input []byte) (authConfigs AuthConfigs, err error) {
	authConfigs = AuthConfigs{}

	r := bytes.NewReader(input)

	cf := configfile.New("")
	if err := cf.LoadFromReader(r); err != nil {
		return nil, err
	}

	// convert from the CLI type to the API type
	for host, conf := range cf.GetAuthConfigs() {
		authConfigs[host] = types.AuthConfig(conf)
	}

	return authConfigs, nil
}

// ExtractBasicAuthForHost will extract the basic auth info for a registry host from an AuthConfigs instance
func ExtractBasicAuthForHost(authConfigs AuthConfigs, host string) (string, string, error) {
	ac, ok := authConfigs[host]
	if !ok {
		var servers []string
		for url := range authConfigs {
			servers = append(servers, url)
		}

		return "", "", fmt.Errorf("registry %q is not in list of registries for this auth source %v", host, servers)
	}

	return ac.Username, ac.Password, nil
}
