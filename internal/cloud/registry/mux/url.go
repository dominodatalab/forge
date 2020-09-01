package mux

import (
	"errors"
	"regexp"

	"github.com/dominodatalab/forge/internal/cloud/registry/types"
)

type schemeMap map[*regexp.Regexp]types.AuthLoader

type URLMux struct {
	schemes schemeMap
}

func NewURLMux() *URLMux {
	return &URLMux{
		schemes: make(schemeMap),
	}
}

func (m *URLMux) RegisterLoader(re *regexp.Regexp, loader types.AuthLoader) {
	m.schemes[re] = loader
}

func (m *URLMux) FromString(url string) (types.AuthLoader, error) {
	for r, loader := range m.schemes {
		if r.MatchString(url) {
			return loader, nil
		}
	}
	return nil, errors.New("no loader found")
}
