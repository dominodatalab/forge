package bkimage

import (
	"fmt"
	"os"
	"path/filepath"

	fuseoverlayfs "github.com/AkihiroSuda/containerd-fuse-overlayfs"
	"github.com/containerd/containerd/content"
	"github.com/containerd/containerd/images"
	"github.com/containerd/containerd/metadata"
	"github.com/containerd/containerd/remotes/docker"
	"github.com/containerd/containerd/snapshots/overlay"
	"github.com/go-logr/logr"
	"github.com/moby/buildkit/control"
	"github.com/moby/buildkit/session"
	"github.com/moby/buildkit/worker/base"
	"github.com/pkg/errors"

	"github.com/dominodatalab/forge/internal/builder/embedded/bkimage/types"
)

// runtime executor
const ociRuntime = "runc"

type Client struct {
	backend string
	rootDir string

	metadataDB   *metadata.DB
	imageStore   images.Store
	contentStore content.Store

	sessionManager *session.Manager
	controller     *control.Controller
	workerOpt      *base.WorkerOpt // NOTE: modified

	// dynamic elements
	registryHosts   docker.RegistryHosts
	hostCredentials CredentialsFn

	logger logr.Logger
}

func NewClient(rootDir, backend string, logger logr.Logger) (*Client, error) {
	// select appropriate system backend
	if backend == types.AutoBackend {
		switch {
		case overlay.Supported(rootDir) == nil:
			backend = types.OverlayFSBackend
		case fuseoverlayfs.Supported(rootDir) == nil:
			backend = types.FuseOverlayFSBackend
		default:
			backend = types.NativeBackend
		}
	}

	logger.Info(fmt.Sprintf("Using filesystem as backend: %s", backend))

	// create working directory
	workDir := filepath.Join(rootDir, ociRuntime, string(backend))
	if err := os.MkdirAll(workDir, 0700); err != nil {
		return nil, err
	}

	// create operational client
	client := &Client{
		backend: backend,
		rootDir: rootDir,
		logger:  logger,
	}
	client.ResetHostConfigurations()

	if err := client.initDataStores(); err != nil {
		return nil, errors.Wrap(err, "initializing data stores failed")
	}

	return client, nil
}

func (c *Client) SetLogger(logger logr.Logger) {
	c.logger = logger
}
