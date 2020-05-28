package builder

import (
	"context"

	"github.com/dominodatalab/forge/internal/builder/config"
	"github.com/dominodatalab/forge/internal/builder/embedded"
)

type OCIImageBuilder interface {
	BuildAndPush(context.Context, *config.BuildOptions) ([]string, error)
}

func New() (OCIImageBuilder, error) {
	return embedded.NewDriver()
}
