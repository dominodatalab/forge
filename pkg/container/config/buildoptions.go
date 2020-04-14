package config

import "time"

type Registry struct {
	URL      string
	Username string
	Password string
	Insecure bool
}

type BuildOptions struct {
	Context   string
	ImageName string
	Registry  Registry
	Labels    map[string]string
	BuildArgs []string
	NoCache   bool
	CpuQuota  uint16
	Memory    string
	Timeout   time.Duration
	SizeLimit uint64
}
