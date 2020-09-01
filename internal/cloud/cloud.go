package cloud

import (
	"github.com/dominodatalab/forge/internal/cloud/registry"
	_ "github.com/dominodatalab/forge/internal/cloud/registry/acr"
	_ "github.com/dominodatalab/forge/internal/cloud/registry/ecr"
	_ "github.com/dominodatalab/forge/internal/cloud/registry/gcr"
)

var RetrieveRegistryAuthorization = registry.RetrieveAuthorization
