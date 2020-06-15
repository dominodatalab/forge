package builder

import (
	"context"
	"github.com/dominodatalab/forge/internal/config"
	"github.com/dominodatalab/forge/plugins/preparer"

	"github.com/dominodatalab/forge/internal/builder/embedded"
)

type OCIImageBuilder interface {
	BuildAndPush(context.Context, *config.BuildOptions) ([]string, error)
}

func New(preparerPlugins []*preparer.Plugin) (OCIImageBuilder, error) {
	return embedded.NewDriver(preparerPlugins)
}
