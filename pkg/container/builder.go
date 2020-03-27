package container

import (
	"context"

	"github.com/dominodatalab/forge/pkg/container/config"
)

type RuntimeBuilder interface {
	Init() error
	Build(ctx context.Context, opts config.BuildOptions) (string, error)
}

func NewBuilder() RuntimeBuilder {
	return nil
	//return runc.NewImageBuilder()
}
