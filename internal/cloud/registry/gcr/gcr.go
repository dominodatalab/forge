package gcr

import (
	"context"
	"errors"
	"regexp"

	"github.com/dominodatalab/forge/internal/cloud/registry/types"
)

var URLRegex = regexp.MustCompile(`^gcr\.io$`)

func LoadAuths(ctx context.Context, url string) (types.AuthConfigs, error) {
	return nil, errors.New("GCR is unsupported")
}
