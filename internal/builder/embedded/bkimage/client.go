package bkimage

import (
	"os"
	"path/filepath"

	"github.com/containerd/containerd/snapshots/overlay"
	"github.com/moby/buildkit/control"
	"github.com/moby/buildkit/session"
	"github.com/moby/buildkit/worker/base"
	log "github.com/sirupsen/logrus"

	"github.com/dominodatalab/forge/internal/builder/embedded/bkimage/types"
)

// runtime executor
const ociRuntime = "runc"

type Client struct {
	backend string
	rootDir string

	sessionManager *session.Manager
	controller     *control.Controller
	workerOpt      *base.WorkerOpt // NOTE: modified
}

func NewClient(rootDir, backend string) (*Client, error) {
	// select appropriate backend
	if backend == types.AutoBackend {
		if overlay.Supported(rootDir) == nil { // NOTE: this operation generates the root directory
			backend = types.OverlayFSBackend
		} else {
			backend = types.NativeBackend
		}
	}
	log.Infof("Using filesystem as backend: %s", backend)

	// create working directory
	workDir := filepath.Join(rootDir, ociRuntime, string(backend))
	if err := os.MkdirAll(workDir, 0700); err != nil {
		return nil, err
	}

	// create client with operational data
	return &Client{
		backend: backend,
		rootDir: rootDir,
	}, nil
}
