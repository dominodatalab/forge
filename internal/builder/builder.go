package builder

import (
	"context"

	"github.com/dominodatalab/forge/internal/builder/config"
	"github.com/dominodatalab/forge/internal/builder/embedded"
	"github.com/go-logr/logr"
)

type OCIImageBuilder interface {
	SetLogger(logr.Logger)
	BuildAndPush(context.Context, *config.BuildOptions) ([]string, error)
}

func New(logger logr.Logger) (OCIImageBuilder, error) {
	return embedded.NewDriver(logger)
}
