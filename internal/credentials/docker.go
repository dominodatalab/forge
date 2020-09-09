package credentials

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/docker/docker/api/types"
	"github.com/pkg/errors"
)

// AuthConfigs is a map of registry urls to authentication credentials.
type AuthConfigs map[string]types.AuthConfig

// DockerConfigJSON models the structure of .dockerconfigfile data.
type DockerConfigJSON struct {
	Auths AuthConfigs `json:"auths"`
}

// ExtractDockerAuth will return a username/password combination from a JSON-representation of a Docker config file.
// This function will process credentials from any of the possible auth-sources within that configuration.
func ExtractDockerAuth(input []byte, host string) (string, string, error) {
	var output DockerConfigJSON
	if err := json.Unmarshal(input, &output); err != nil {
		return "", "", errors.Wrap(err, "cannot parse docker config contents")
	}

	auth, ok := output.Auths[host]
	if !ok {
		var servers []string
		for url := range output.Auths {
			servers = append(servers, url)
		}

		return "", "", fmt.Errorf("registry %q is not in server list %v", host, servers)
	}

	switch {
	case auth.Username != "" && auth.Password != "":
		return auth.Username, auth.Password, nil
	case auth.Auth != "":
		token, err := base64.StdEncoding.DecodeString(auth.Auth)
		if err != nil {
			return "", "", errors.Wrap(err, "invalid docker authorization token")
		}

		up := strings.Split(string(token), ":")
		if len(up) != 2 {
			return "", "", errors.Wrapf(err, "invalid docker username/password: %q", up)
		}
		return up[0], up[1], nil
	default:
		return "", "", fmt.Errorf("cannot extract auth from config: %+v", auth)
	}
}
