package cloud

import (
	"context"
	"regexp"
	"testing"

	"github.com/docker/docker/api/types"
)

func TestRegistry_RetrieveAuthorization(t *testing.T) {
	expected := &types.AuthConfig{
		Username: "test-user",
		Password: "test-pass",
	}
	registry := &Registry{}
	registry.Register(regexp.MustCompile(`^my.cloud`), func(ctx context.Context, url string) (*types.AuthConfig, error) {
		return expected, nil
	})

	ctx := context.Background()
	auth, err := registry.RetrieveAuthorization(ctx, "my.cloud/best/cloud")
	if err != nil {
		t.Errorf("unexpected err: %v", err)
	}
	if auth != expected {
		t.Errorf("wrong auth: got %v, want %v", auth, expected)
	}

	auth, err = registry.RetrieveAuthorization(ctx, "your.cloud/silly/cloud")
	if err != errNoLoader {
		t.Errorf("wrong err: got %v, want %v", err, errNoLoader)
	}
	if auth != nil {
		t.Errorf("unexpected auth: got %v", auth)
	}
}
