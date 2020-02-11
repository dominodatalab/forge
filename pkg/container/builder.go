package container

import (
	"context"

	"github.com/dominodatalab/forge/api/v1alpha1"
)

type RuntimeBuilder interface {
	Build(ctx context.Context, spec v1alpha1.ContainerImageBuildSpec) (string, error)
}
