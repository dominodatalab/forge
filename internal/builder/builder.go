package builder

import (
	"context"

	"github.com/dominodatalab/forge/internal/builder/embedded"
	"github.com/dominodatalab/forge/internal/config"
	"github.com/dominodatalab/forge/plugins/preparer"
	"github.com/go-logr/logr"
)

type OCIImageBuilder interface {
	SetLogger(logr.Logger)
	BuildAndPush(context.Context, *config.BuildOptions) ([]string, error)
}

func New(preparerPlugins []*preparer.Plugin, cacheImageLayers bool, logger logr.Logger) (OCIImageBuilder, error) {
	return embedded.NewDriver(preparerPlugins, cacheImageLayers, logger)
}
