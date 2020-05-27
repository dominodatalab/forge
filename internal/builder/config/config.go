package config

import "time"

type Registry struct {
	Host     string
	Username string
	Password string
	NonSSL   bool
}

type BuildOptions struct {
	ContextURL     string
	ImageName      string
	ImageSizeLimit uint64
	Labels         map[string]string
	BuildArgs      []string
	NoCache        bool
	Timeout        time.Duration
	Registries     []Registry

	// NOTE: these are not currently used; remove them?
	CpuQuota uint16
	Memory   string
}
