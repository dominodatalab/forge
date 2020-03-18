package config

type BuildOptions struct {
	Context          string
	ImageName        string
	RegistryURL      string
	InsecureRegistry bool
	Labels           map[string]string
	BuildArgs        []string
	NoCache          bool
	CpuQuota         uint16
	Memory           string
}
