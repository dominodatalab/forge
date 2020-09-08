package mux

import (
	"context"
	"regexp"

	"github.com/docker/docker/api/types"
)

// AuthLoader is function that must be implemented by every cloud-specific registry authorization provider.
type AuthLoader func(ctx context.Context, url string) (*types.AuthConfig, error)

type schemeMap map[*regexp.Regexp]AuthLoader

// URLMux provides a means of multiplexing cloud registry authorization provides based on a URL.
type URLMux struct {
	schemes schemeMap
}

// NewURLMux returns an initialized form of the URLMux.
func NewURLMux() *URLMux {
	return &URLMux{
		schemes: make(schemeMap),
	}
}

// RegisterLoader will create a new url regex -> authorization loader scheme.
func (m *URLMux) RegisterLoader(re *regexp.Regexp, loader AuthLoader) {
	m.schemes[re] = loader
}

// FromString retrieves and authorization loader for a given url. An error is returned if no matching loader is found.
func (m *URLMux) FromString(url string) (AuthLoader, error) {
	for r, loader := range m.schemes {
		if r.MatchString(url) {
			return loader, nil
		}
	}
	return nil, NoLoaderFoundError{URL: url}
}
