package bkimage

import (
	"crypto/tls"
	"crypto/x509"
	"io/ioutil"
	"net"
	"net/http"
	"time"

	"github.com/containerd/containerd/remotes/docker"
	"github.com/moby/buildkit/util/tracing"
	"github.com/pkg/errors"
)

type CredentialsFn func(host string) (username string, password string, err error)
type TLSEnabledFn func(host string) (enabled bool, err error)

func (c *Client) ResetHostConfigurations() {
	c.registryHosts = docker.ConfigureDefaultRegistries()
	c.hostCredentials = func(host string) (username string, password string, err error) {
		return "", "", nil
	}
}

func (c *Client) ConfigureHosts(hostCredentials CredentialsFn, matchNonSSL TLSEnabledFn, rootCA string) error {
	transport := newDefaultTransport()

	tlsConfig := &tls.Config{}
	if rootCA != "" {
		dt, err := ioutil.ReadFile(rootCA)
		if err != nil {
			return errors.Wrapf(err, "failed to read root CA from %q", rootCA)
		}

		systemPool, err := x509.SystemCertPool()
		if err != nil {
			return errors.Wrapf(err, "failed to initialize system CA pool")
		}

		tlsConfig.RootCAs = systemPool
		tlsConfig.RootCAs.AppendCertsFromPEM(dt)
	}

	transport.TLSClientConfig = tlsConfig
	client := &http.Client{
		Transport: tracing.NewTransport(transport),
	}

	authorizer := docker.NewDockerAuthorizer(
		docker.WithAuthCreds(hostCredentials),
		docker.WithAuthClient(client),
	)

	c.hostCredentials = hostCredentials
	c.registryHosts = docker.ConfigureDefaultRegistries(
		docker.WithAuthorizer(authorizer),
		docker.WithPlainHTTP(matchNonSSL),
		docker.WithClient(client),
	)

	return nil
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

// This is 1:1 copy of https://github.com/moby/buildkit/blob/master/util/resolver/resolver.go#L204-L219
// that we need to ensure we can add the correct CA certificate configuration.
func newDefaultTransport() *http.Transport {
	return &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
			DualStack: true,
		}).DialContext,
		MaxIdleConns:          10,
		IdleConnTimeout:       30 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 5 * time.Second,
		DisableKeepAlives:     true,
		TLSNextProto:          make(map[string]func(authority string, c *tls.Conn) http.RoundTripper),
	}
}
