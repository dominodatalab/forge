package bkimage

import (
	"fmt"
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
	workDir := filepath.Join(rootDir, ociRuntime, backend)

	autoSelectFn := func() string {
		if err := overlay.Supported(workDir); err != nil {
			logger.Info("overlayfs not unsupported", "error", err)
		} else {
			return types.OverlayFSBackend
		}

		if err := fuseoverlayfs.Supported(workDir); err != nil {
			logger.Info("fuse-overlayfs not supported", "error", err)
		} else {
			return types.FuseOverlayFSBackend
		}

		return types.NativeBackend
	}

	// select appropriate system backend
	if backend == types.AutoBackend {
		backend = autoSelectFn()
	}

	logger.Info(fmt.Sprintf("Using filesystem as backend: %s", backend))

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
