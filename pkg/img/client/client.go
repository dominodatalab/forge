package client

import (
	"os"
	"path/filepath"

	"github.com/containerd/containerd/snapshots/overlay"
	"github.com/moby/buildkit/control"
	"github.com/moby/buildkit/session"
	"github.com/moby/buildkit/worker/base"
	"github.com/sirupsen/logrus"

	"github.com/dominodatalab/forge/pkg/img/types"
)

// Client holds the information for the client we will use for communicating
// with the buildkit controller.
type Client struct {
	backend string
	root    string

	sessionManager *session.Manager
	controller     *control.Controller
	workerOpt      *base.WorkerOpt
}

// New returns a new client for communicating with the buildkit controller.
func New(root, backend string) (*Client, error) {
	// Set the name for the directory executor.
	name := "runc"

	switch backend {
	case types.AutoBackend:
		if overlay.Supported(root) == nil {
			backend = types.OverlayFSBackend
		} else {
			backend = types.NativeBackend
		}
		logrus.Debugf("using backend: %s", backend)
	}

	if backend == types.NativeBackend {
		logrus.Warn("Using native fs backend for image building, performance may be severely impacted")
	}

	// Create the root/
	root = filepath.Join(root, name, backend)
	if err := os.MkdirAll(root, 0700); err != nil {
		return nil, err
	}

	// Create the start of the client.
	return &Client{
		backend: backend,
		root:    root,
	}, nil
}

// Close safely closes the client.
// This used to shut down the FUSE server but since that was removed
// it is basically a no-op now.
func (c *Client) Close() {}
