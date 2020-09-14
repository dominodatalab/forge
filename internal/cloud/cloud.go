package cloud

import (
	"github.com/dominodatalab/forge/internal/cloud/registry"
	_ "github.com/dominodatalab/forge/internal/cloud/registry/ecr"
	"github.com/dominodatalab/forge/internal/cloud/registry/mux"
)

var (
	RetrieveRegistryAuthorization = registry.RetrieveAuthorization
	IsNoRegistryAuthLoaderFound   = mux.IsNoLoaderFound
)
