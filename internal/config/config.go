package config

import (
	"path/filepath"
	"time"
)

const (
	// DynamicCredentialsPath is the directory inside the build job where dynamic cloud registry credentials are stored.
	DynamicCredentialsPath = "/tmp/docker"
	// DynamicCredentialsFilename is the name of the file housing dynamic cloud registry credentials.
	DynamicCredentialsFilename = "config.json"
)

// DynamicCredentialsFilepath is the full path to the dynamic cloud registry credentials.
var DynamicCredentialsFilepath = filepath.Join(DynamicCredentialsPath, DynamicCredentialsFilename)

type Registry struct {
	Host     string
	NonSSL   bool
	Username string
	Password string
}

type BuildOptions struct {
	ContextURL              string
	ContextTimeout          time.Duration
	ImageName               string
	ImageSizeLimit          uint64
	Labels                  map[string]string
	BuildArgs               []string
	DisableBuildCache       bool
	DisableLayerCacheExport bool
	Timeout                 time.Duration
	Registries              []Registry
	PushRegistries          []string
	PluginData              map[string]string
	CacheFrom               []string
}
