package config

type Registry struct {
	ServerURL string
	Username  string
	Password  string
	Insecure  bool
}

type Image struct {
	Name      string
	BuildArgs []string
	Commands  []string
}

type BuildOptions struct {
	NoCache bool
	Image
	Registry
}
