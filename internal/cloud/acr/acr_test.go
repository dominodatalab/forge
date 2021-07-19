package acr

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/Azure/azure-sdk-for-go/services/preview/containerregistry/runtime/2019-08-15-preview/containerregistry"
	"github.com/Azure/go-autorest/autorest/azure/auth"
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/dominodatalab/forge/internal/cloud"
	"github.com/go-logr/logr"
	"github.com/go-logr/zapr"
	"go.uber.org/zap"
)

func TestRegister(t *testing.T) {
	if os.Getenv(auth.ClientSecret) == "" {
		t.Skip("Skipping, azure not setup")
	}

	registry := &cloud.Registry{}

	err := Register(newLogger(t), registry)
	if err != nil {
		t.Error(err)
	}
}

func TestRegisterNoSecret(t *testing.T) {
	secret := os.Getenv(auth.ClientSecret)
	os.Unsetenv(auth.ClientSecret)
	defer func() {
		os.Setenv(auth.ClientSecret, secret)
	}()

	registry := &cloud.Registry{}
	err := Register(newLogger(t), registry)
	if err == nil {
		t.Error("expecting an error")
	} else if !strings.Contains(err.Error(), "MSI not available") {
		t.Errorf("wrong error: %v", err)
	}
}

func TestAuthenticate(t *testing.T) {
	if os.Getenv(auth.ClientSecret) == "" {
		t.Skip("Skipping, azure not setup")
	}

	acrRegistry := os.Getenv("ACR_REGISTRY")
	if len(acrRegistry) == 0 {
		t.Fatal("must set ACR_REGISTRY environment variable")
	}

	p, err := newProvider(newLogger(t))
	if err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	t.Run("invalid url", func(t *testing.T) {
		auth, err := p.authenticate(ctx, "bogus.azure.io")
		if auth != nil {
			t.Errorf("auth not nil")
		}
		if !strings.HasPrefix(err.Error(), "invalid acr url") {
			t.Fatalf("wrong error: %s", err)
		}
	})

	t.Run("valid url", func(t *testing.T) {
		auth, err := p.authenticate(ctx, acrRegistry)
		if err != nil {
			t.Fatalf("%#v", err)
		}
		if len(auth.Username) == 0 || len(auth.Password) == 0 {
			t.Fatalf("incorrect auth config: %v", auth)
		}

		// verify we can obtain an access token from the refresh token
		accessClient := containerregistry.NewAccessTokensClient("https://" + acrRegistry)
		r, err := accessClient.Get(ctx, acrRegistry, "repository:repo:pull,push", auth.Password)
		if err != nil {
			t.Fatal(err)
		}
		if a := to.String(r.AccessToken); len(a) == 0 {
			t.Fatalf("invalid access token: %v", r)
		}
	})
}

func newLogger(t *testing.T) logr.Logger {
	zlog, err := zap.NewDevelopment()
	if err != nil {
		t.Fatal(err)
	}
	return zapr.NewLogger(zlog)
}
