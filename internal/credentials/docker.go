package credentials

type DockerConfig map[string]DockerConfigEntry

type DockerConfigEntry struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// DockerConfigJSON models the structure of .dockerconfigfile data
type DockerConfigJSON struct {
	Auths DockerConfig `json:"auths"`
}
