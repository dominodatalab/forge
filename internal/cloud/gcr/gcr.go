package gcr

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/dominodatalab/forge/internal/cloud"
	"github.com/go-logr/logr"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

const cloudPlatformScope = "https://www.googleapis.com/auth/cloud-platform"

var (
	gcrRegex      = regexp.MustCompile(`.*-docker\.pkg\.dev|(?:.*\.)?gcr\.io`)
	defaultClient = &http.Client{
		Transport: &http.Transport{
			Dial: (&net.Dialer{
				Timeout:   2 * time.Second,
				KeepAlive: 30 * time.Second,
			}).Dial,
		},
	}
)

type gcrProvider struct {
	logger      logr.Logger
	tokenSource oauth2.TokenSource
}

func Register(ctx context.Context, logger logr.Logger, registry *cloud.Registry) error {
	provider, err := newProvider(ctx, logger)
	if err != nil {
		logger.Info("GCR not registered", "error", err)
		if strings.Contains(err.Error(), "could not find default credentials") {
			return nil
		}
		return err
	}

	registry.Register(gcrRegex, provider.authenticate)
	logger.Info("GCR registered")
	return nil
}

func newProvider(ctx context.Context, logger logr.Logger) (*gcrProvider, error) {
	creds, err := google.FindDefaultCredentials(ctx, cloudPlatformScope)
	if err != nil {
		return nil, err
	}

	return &gcrProvider{logger: logger.WithName("gcrProvider"), tokenSource: creds.TokenSource}, nil
}

func (g *gcrProvider) authenticate(ctx context.Context, server string) (*types.AuthConfig, error) {
	match := gcrRegex.FindAllString(server, -1)
	if len(match) != 1 {
		return nil, fmt.Errorf("invalid gcr url: %q should match %v", server, gcrRegex)
	}

	token, err := g.tokenSource.Token()
	if err != nil {
		return nil, err
	}

	loginServerURL := "https://" + match[0]
	directive, err := cloud.ChallengeLoginServer(ctx, loginServerURL)
	if err != nil {
		return nil, err
	}

	// obtain the registry token
	req, err := http.NewRequestWithContext(ctx, "GET", directive.Realm, nil)
	if err != nil {
		return nil, err
	}

	v := url.Values{}
	v.Set("service", directive.Service)
	v.Set("client_id", "forge")
	req.URL.RawQuery = v.Encode()
	req.URL.User = url.UserPassword("oauth2accesstoken", token.AccessToken)
	resp, err := defaultClient.Do(req)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()
	content, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("failed to obtain token:\n %s", content)
	}

	type tokenResponse struct {
		Token        string `json:"token"`
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
	}

	var response tokenResponse
	if err := json.Unmarshal(content, &response); err != nil {
		return nil, err
	}

	// Some registries set access_token instead of token.
	if response.AccessToken != "" {
		response.Token = response.AccessToken
	}

	// Find a token to turn into a Bearer authenticator
	if response.Token == "" {
		return nil, fmt.Errorf("no token in bearer response:\n%s", content)
	}

	// buildkit only supports username/password
	return &types.AuthConfig{
		Username: "oauth2accesstoken",
		Password: token.AccessToken,

		RegistryToken: response.Token,
	}, nil
}
