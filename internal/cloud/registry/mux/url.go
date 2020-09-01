package mux

import (
	"fmt"
	"regexp"

	"github.com/dominodatalab/forge/internal/cloud/registry/types"
)

type schemeMap map[*regexp.Regexp]types.AuthLoader

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
func (m *URLMux) RegisterLoader(re *regexp.Regexp, loader types.AuthLoader) {
	m.schemes[re] = loader
}

// FromString retrieves and authorization loader for a given url. An error is returned if no matching loader is found.
func (m *URLMux) FromString(url string) (types.AuthLoader, error) {
	for r, loader := range m.schemes {
		if r.MatchString(url) {
			return loader, nil
		}
	}
	return nil, fmt.Errorf("no loader found for %q", url)
}
