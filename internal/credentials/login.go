package credentials

import (
	"context"
	"fmt"

	"github.com/docker/cli/cli/config"
	"github.com/docker/cli/cli/config/configfile"
	"github.com/docker/cli/cli/config/types"
	dockerapitypes "github.com/docker/docker/api/types"
	"github.com/docker/docker/registry"
	registryapi "github.com/genuinetools/reg/registry"
	"github.com/sirupsen/logrus"
)

func DockerLogin(username, password, server string, nonSSL bool) error {
	ctx := context.TODO()

	dockerCfg, authConfig, err := configureAuth(username, password, server)
	if err != nil {
		return err
	}

	apiAuth := dockerapitypes.AuthConfig{
		Username:      authConfig.Username,
		Password:      authConfig.Password,
		Auth:          authConfig.Auth,
		Email:         authConfig.Email,
		ServerAddress: authConfig.ServerAddress,
		IdentityToken: authConfig.IdentityToken,
		RegistryToken: authConfig.RegistryToken,
	}
	client, err := registryapi.New(ctx, apiAuth, registryapi.Opt{Debug: true, NonSSL: nonSSL})
	if err != nil {
		return fmt.Errorf("creating registry client failed: %w", err)
	}
	token, err := client.Token(ctx, client.URL)
	if err != nil {
		return fmt.Errorf("failed to get registry token: %w", err)
	}
	if token != "" {
		authConfig.Password = ""
		authConfig.IdentityToken = token
	}

	if err := dockerCfg.GetCredentialsStore(authConfig.ServerAddress).Store(*authConfig); err != nil {
		return fmt.Errorf("saving credentials failed: %w", err)
	}
	logrus.Infof("Successfully logged into %s", server)

	return nil
}

func configureAuth(username, password, server string) (*configfile.ConfigFile, *types.AuthConfig, error) {
	serverAddr := registry.ConvertToHostname(server)

	dockerCfg, err := config.Load(config.Dir())
	if err != nil {
		return nil, nil, fmt.Errorf("loading docker config failed: %w", err)
	}
	authConfig, err := dockerCfg.GetAuthConfig(serverAddr)
	if err != nil {
		return nil, nil, fmt.Errorf("getting auth config for %s failed: %w", serverAddr, err)
	}

	if dockerCfg.CredentialHelpers[serverAddr] != "" && authConfig.Username != "" && authConfig.Password != "" {
		return dockerCfg, &authConfig, nil
	}
	authConfig.ServerAddress = serverAddr
	authConfig.Username = username
	authConfig.Password = password

	return dockerCfg, &authConfig, nil
}
