package gcr

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"testing"

	"github.com/dominodatalab/forge/internal/cloud"
	"github.com/go-logr/logr"
	"github.com/go-logr/zapr"
	"go.uber.org/zap"
)

var env_credentials = "GOOGLE_APPLICATION_CREDENTIALS"

func TestRegister(t *testing.T) {
	if os.Getenv(env_credentials) == "" {
		t.Skip("Skipping, gcp not setup")
	}

	registry := &cloud.Registry{}

	err := Register(context.TODO(), newLogger(t), registry)
	if err != nil {
		t.Error(err)
	}
}

func TestRegisterNoCredentials(t *testing.T) {
	secret := os.Getenv(env_credentials)
	os.Unsetenv(env_credentials)
	t.Cleanup(func() {
		os.Setenv(env_credentials, secret)
	})

	registry := &cloud.Registry{}
	err := Register(context.TODO(), newLogger(t), registry)
	if err != nil {
		t.Error(err)
	}
}

func TestAuthenticate(t *testing.T) {
	if os.Getenv(env_credentials) == "" {
		t.Skip("Skipping, gcp not setup")
	}

	p, err := newProvider(context.TODO(), newLogger(t))
	if err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	t.Run("invalid url", func(t *testing.T) {
		auth, err := p.authenticate(ctx, "bogus.g.io")
		if err == nil {
			t.Error("unexpected error")
		}
		if auth != nil {
			t.Errorf("auth not nil")
		}
	})

	t.Run("valid url", func(t *testing.T) {
		for _, tt := range []struct {
			name string
		}{
			{"gcr.io"},
			{"us-west1-docker.pkg.dev"},
		} {
			t.Run(tt.name, func(t *testing.T) {
				auth, err := p.authenticate(ctx, tt.name)
				if err != nil {
					t.Fatalf("%#v", err)
				}
				if auth.Username == "" || auth.Password == "" || auth.RegistryToken == "" {
					t.Fatalf("incorrect auth config: %v", auth)
				}

				// verify the registry token
				req, err := http.NewRequestWithContext(ctx, "GET", "https://"+tt.name+"/v2/", nil)
				if err != nil {
					t.Fatal(err)
				}
				req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", auth.RegistryToken))
				resp, err := defaultClient.Do(req)
				if err != nil {
					t.Fatal(err)
				}
				defer resp.Body.Close()

				if resp.StatusCode != 200 {
					t.Fatalf("non 200 status code: %d", resp.StatusCode)
				}
			})
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
