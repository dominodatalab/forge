package container

import (
	"context"

	"github.com/dominodatalab/forge/pkg/container/config"
	"github.com/dominodatalab/forge/pkg/container/runc"
)

type RuntimeBuilder interface {
	//Initialize(ctx context.Context) error
	Build(ctx context.Context, opts config.BuildOptions) (string, error)
}

func NewBuilder() (RuntimeBuilder, error) {
	hostURL, err := runc.EnsureBuildkitDaemon()
	if err != nil {
		return nil, err
	}

	return runc.NewRuncBuilder(hostURL)
}
