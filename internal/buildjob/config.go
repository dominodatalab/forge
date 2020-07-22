package buildjob

import "github.com/dominodatalab/forge/internal/message"

type Config struct {
	ResourceName        string
	BrokerOpts          *message.Options
	PreparerPluginsPath string
	EnableLayerCaching  bool
	Debug               bool
}
