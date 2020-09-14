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

// ExtractDockerAuth will return a username/password combination from a JSON-representation of a Docker config file.
func ExtractDockerAuth(input []byte, host string) (string, string, error) {
	r := bytes.NewReader(input)

	cf := configfile.New("")
	if err := cf.LoadFromReader(r); err != nil {
		return "", "", err
	}

	ac, ok := cf.GetAuthConfigs()[host]
	if !ok {
		var servers []string
		for url := range cf.GetAuthConfigs() {
			servers = append(servers, url)
		}

		return "", "", fmt.Errorf("registry %q is not in server list %v", host, servers)
	}

	return ac.Username, ac.Password, nil
}
