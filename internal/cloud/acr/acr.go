package acr

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"time"

	"github.com/Azure/azure-sdk-for-go/services/preview/containerregistry/runtime/2019-08-15-preview/containerregistry"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/adal"
	"github.com/Azure/go-autorest/autorest/azure/auth"
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/docker/docker/api/types"
	"github.com/dominodatalab/forge/internal/cloud"
	"github.com/go-logr/logr"
)

// https://github.com/Azure/acr/blob/main/docs/AAD-OAuth.md

const acrUserForRefreshToken = "00000000-0000-0000-0000-000000000000"

var (
	acrRegex = regexp.MustCompile(`.*\.azurecr\.io|.*\.azurecr\.cn|.*\.azurecr\.de|.*\.azurecr\.us`)

	noCredsErr = errors.New("no Azure Credentials")
)

type acrProvider struct {
	logger                logr.Logger
	tenantID              string
	servicePrincipalToken *adal.ServicePrincipalToken
}

func Register(logger logr.Logger, registry *cloud.Registry) error {
	provider, err := newProvider(logger)
	if err != nil {
		logger.Info("ACR not registered", "error", err)
		if err == noCredsErr {
			return nil
		}
		return err
	}

	registry.Register(acrRegex, provider.authenticate)
	logger.Info("ACR registered")
	return nil
}

func newProvider(logger logr.Logger) (*acrProvider, error) {
	settings, err := auth.GetSettingsFromEnvironment()
	if err != nil {
		return nil, err
	}

	// the minimum set of required values
	if settings.Values[auth.TenantID] == "" || settings.Values[auth.ClientID] == "" {
		return nil, noCredsErr
	}

	var spt *adal.ServicePrincipalToken
	if cc, err := settings.GetClientCredentials(); err == nil {
		if spt, err = cc.ServicePrincipalToken(); err != nil {
			return nil, err
		}
	} else {
		ctx := context.Background()
		for i := 0; i < 3; i++ {
			if spt, err = settings.GetMSI().ServicePrincipalToken(); err == nil {
				break
			}
			logger.Error(err, "retrying", "attempt", i+1)
			if !autorest.DelayForBackoff(time.Second, i, ctx.Done()) {
				return nil, ctx.Err()
			}
		}
		if err != nil {
			// IMDS can take some time to setup, restart the process
			return nil, fmt.Errorf("retreiving Service Principal Token from MSI failed: %w", err)
		}
	}

	return &acrProvider{logger: logger.WithName("acrProvider"), tenantID: settings.Values[auth.TenantID], servicePrincipalToken: spt}, nil
}

func (a *acrProvider) authenticate(ctx context.Context, server string) (*types.AuthConfig, error) {
	match := acrRegex.FindAllString(server, -1)
	if len(match) != 1 {
		return nil, fmt.Errorf("invalid acr url: %q should match %v", server, acrRegex)
	}

	loginServer := match[0]
	if err := a.servicePrincipalToken.EnsureFreshWithContext(ctx); err != nil {
		return nil, err
	}

	armAccessToken := a.servicePrincipalToken.OAuthToken()
	loginServerURL := "https://" + loginServer
	directive, err := cloud.ChallengeLoginServer(ctx, loginServerURL)
	if err != nil {
		return nil, err
	}

	refreshClient := containerregistry.NewRefreshTokensClient(loginServerURL)
	refreshToken, err := refreshClient.GetFromExchange(ctx, "access_token", directive.Service, a.tenantID, "", armAccessToken)
	if err != nil {
		return nil, err
	}

	return &types.AuthConfig{
		Username: acrUserForRefreshToken,
		Password: to.String(refreshToken.RefreshToken),
	}, nil
}
