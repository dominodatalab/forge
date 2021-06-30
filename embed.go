package forge

import "embed"

//go:embed config/crd/bases/*.yaml
var CRDs embed.FS
