package cloud

import (
	"context"
	"errors"
	"regexp"

	"github.com/docker/docker/api/types"
)

var errNoLoader = errors.New("no loader found")

type AuthLoader func(ctx context.Context, server string) (*types.AuthConfig, error)

type Registry struct {
	loaders map[*regexp.Regexp]AuthLoader
}

// RetrieveAuthorization will multiplex registered auth loaders based on url pattern and use the appropriate one to
// make an authorization request. The returned value can be marshalled into the contents of a Docker config.json file.
func (r *Registry) RetrieveAuthorization(ctx context.Context, server string) (*types.AuthConfig, error) {
	for r, loader := range r.loaders {
		if r.MatchString(server) {
			return loader(ctx, server)
		}
	}
	return nil, errNoLoader
}

// RegisterLoader will create a new url regex -> authorization loader scheme.
func (r *Registry) Register(re *regexp.Regexp, loader AuthLoader) {
	if r.loaders == nil {
		r.loaders = map[*regexp.Regexp]AuthLoader{}
	}
	r.loaders[re] = loader
}
