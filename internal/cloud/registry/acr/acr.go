package acr

import (
	"context"
	"errors"

	"github.com/docker/docker/api/types"
)

// AuthN details: https://docs.microsoft.com/en-us/azure/container-registry/container-registry-authentication

//var urlRegex = regexp.MustCompile(`^acr\.io$`)

func LoadAuths(ctx context.Context, url string) (*types.AuthConfig, error) {
	return nil, errors.New("GCR is unsupported")
}

//func init() {
//	registry.DefaultURLMux().RegisterLoader(urlRegex, LoadAuths)
//}
