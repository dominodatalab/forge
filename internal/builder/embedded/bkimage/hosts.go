package bkimage

import "github.com/containerd/containerd/remotes/docker"

type CredentialsFn func(host string) (username string, password string, err error)
type TLSEnabledFn func(host string) (enabled bool, err error)

func (c *Client) ResetHostConfigurations() {
	c.registryHosts = docker.ConfigureDefaultRegistries()
	c.hostCredentials = func(host string) (username string, password string, err error) {
		return "", "", nil
	}
}

func (c *Client) ConfigureHosts(hostCredentials CredentialsFn, matchNonSSL TLSEnabledFn) {
	authOpt := docker.WithAuthCreds(hostCredentials)
	authorizer := docker.NewDockerAuthorizer(authOpt)

	c.hostCredentials = hostCredentials
	c.registryHosts = docker.ConfigureDefaultRegistries(
		docker.WithAuthorizer(authorizer),
		docker.WithPlainHTTP(matchNonSSL),
	)
}

func (c *Client) getHostCredentials() CredentialsFn {
	return func(host string) (string, string, error) {
		return c.hostCredentials(host)
	}
}

func (c *Client) getRegistryHosts() docker.RegistryHosts {
	return func(s string) ([]docker.RegistryHost, error) {
		return c.registryHosts(s)
	}
}
