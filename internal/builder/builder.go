package builder

import (
	"context"
	"io"

	"github.com/dominodatalab/forge/internal/builder/config"
	"github.com/dominodatalab/forge/internal/builder/embedded"
)

type OCIImageBuilder interface {
	BuildAndPush(context.Context, *config.BuildOptions, io.Writer) ([]string, error)
}

func New() (OCIImageBuilder, error) {
	return embedded.NewDriver()
}
