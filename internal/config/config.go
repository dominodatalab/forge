package config

import "time"

type Registry struct {
	Host     string
	NonSSL   bool
	Username string
	Password string
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
	PushRegistries []string
	PluginData     map[string]string
}
