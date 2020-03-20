package container

import (
	"context"

	"github.com/dominodatalab/forge/pkg/container/config"
	"github.com/dominodatalab/forge/pkg/container/runc"
)

type RuntimeBuilder interface {
	Init() error
	Build(ctx context.Context, opts config.BuildOptions) (string, error)
}

func NewBuilder() RuntimeBuilder {
	return runc.NewImageBuilder()
}
