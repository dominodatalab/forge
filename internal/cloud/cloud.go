package cloud

import (
	"github.com/dominodatalab/forge/internal/cloud/registry"
	_ "github.com/dominodatalab/forge/internal/cloud/registry/acr"
	_ "github.com/dominodatalab/forge/internal/cloud/registry/ecr"
	_ "github.com/dominodatalab/forge/internal/cloud/registry/gcr"
	"github.com/dominodatalab/forge/internal/cloud/registry/mux"
)

var (
	RetrieveRegistryAuthorization = registry.RetrieveAuthorization
	IsNoRegistryAuthLoaderFound   = mux.IsNoLoaderFound
)
