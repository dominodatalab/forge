package registry

import (
	"context"

	"github.com/dominodatalab/forge/internal/cloud/registry/mux"
	"github.com/dominodatalab/forge/internal/cloud/registry/types"
)

var defaultURLMux = mux.NewURLMux()

func DefaultURLMux() *mux.URLMux {
	return defaultURLMux
}

// RetrieveAuthorization will multiplex registered auth loaders based on url pattern and use the appropriate one to
// make an authorization request. The returned value can be marshalled into the contents of a Docker config.json file.
func RetrieveAuthorization(ctx context.Context, url string) (types.AuthConfigs, error) {
	loader, err := defaultURLMux.FromString(url)
	if err != nil {
		return nil, err
	}
	return loader(ctx, url)
}
