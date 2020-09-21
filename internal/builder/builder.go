package builder

import (
	"context"

	"github.com/dominodatalab/forge/internal/builder/embedded"
	"github.com/dominodatalab/forge/internal/builder/types"
	"github.com/dominodatalab/forge/internal/config"
	"github.com/dominodatalab/forge/plugins/preparer"
	"github.com/go-logr/logr"
)

type OCIImageBuilder interface {
	SetLogger(logr.Logger)
	BuildAndPush(context.Context, *config.BuildOptions) (*types.Image, error)
}

func New(preparerPlugins []*preparer.Plugin, cacheImageLayers bool, logger logr.Logger) (OCIImageBuilder, error) {
	return embedded.NewDriver(preparerPlugins, cacheImageLayers, logger)
}
