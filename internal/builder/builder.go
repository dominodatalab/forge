package builder

import (
	"context"
	bkclient "github.com/moby/buildkit/client"

	"github.com/dominodatalab/forge/internal/builder/config"
	"github.com/dominodatalab/forge/internal/builder/embedded"
)


type OCIImageBuilder interface {
	BuildAndPush(context.Context, *config.BuildOptions, func(chan *bkclient.SolveStatus) error) ([]string, error)
}

func New() (OCIImageBuilder, error) {
	return embedded.NewDriver()
}
