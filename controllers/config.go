package controllers

import "github.com/dominodatalab/forge/internal/message"

type Config struct {
	BuildJobImage        string
	Namespace            string
	MetricsAddr          string
	EnableLeaderElection bool
	BrokerOpts           *message.Options
	PreparerPluginsPath  string
	EnableLayerCaching   bool
	Debug                bool
}
