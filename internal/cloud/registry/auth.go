package registry

import (
	"context"

	"github.com/docker/docker/api/types"

	"github.com/dominodatalab/forge/internal/cloud/registry/mux"
)

var defaultURLMux = mux.NewURLMux()

// DefaultURLMux returns a handle to the url multiplexer.
func DefaultURLMux() *mux.URLMux {
	return defaultURLMux
}

// RetrieveAuthorization will multiplex registered auth loaders based on url pattern and use the appropriate one to
// make an authorization request. The returned value can be marshalled into the contents of a Docker config.json file.
func RetrieveAuthorization(ctx context.Context, url string) (*types.AuthConfig, error) {
	loader, err := defaultURLMux.FromString(url)
	if err != nil {
		return nil, err
	}
	return loader(ctx, url)
}
