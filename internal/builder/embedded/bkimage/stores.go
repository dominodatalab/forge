package bkimage

import (
	"context"
	"fmt"
	"path/filepath"

	fuseoverlayfs "github.com/AkihiroSuda/containerd-fuse-overlayfs"
	"github.com/containerd/containerd/content/local"
	ctdmetadata "github.com/containerd/containerd/metadata"
	"github.com/containerd/containerd/snapshots"
	"github.com/containerd/containerd/snapshots/native"
	"github.com/containerd/containerd/snapshots/overlay"
	containerdsnapshot "github.com/moby/buildkit/snapshot/containerd"
	"github.com/pkg/errors"
	bolt "go.etcd.io/bbolt"

	"github.com/dominodatalab/forge/internal/builder/embedded/bkimage/types"
)

func (c *Client) initDataStores() error {
	// containerd metadata database
	db, err := bolt.Open(filepath.Join(c.rootDir, "container-metadata.db"), 0644, nil)
	if err != nil {
		return err
	}

	cs, err := local.NewStore(filepath.Join(c.rootDir, "content"))
	if err != nil {
		return err
	}

	snapshotDir := filepath.Join(c.rootDir, "snapshotter")
	var snapshotter snapshots.Snapshotter
	switch c.backend {
	case types.NativeBackend:
		snapshotter, err = native.NewSnapshotter(snapshotDir)
	case types.OverlayFSBackend:
		snapshotter, err = overlay.NewSnapshotter(snapshotDir)
	case types.FuseOverlayFSBackend:
		snapshotter, err = fuseoverlayfs.NewSnapshotter(snapshotDir)
	default:
		return fmt.Errorf("%s is not a valid snapshotter", c.backend)
	}
	if err != nil {
		return errors.Wrapf(err, "creating %s snapshotter failed", c.backend)
	}

	metadataDB := ctdmetadata.NewDB(db, cs, map[string]snapshots.Snapshotter{
		c.backend: snapshotter,
	})
	if err := metadataDB.Init(context.Background()); err != nil {
		return err
	}
	c.metadataDB = metadataDB

	// metadata stores
	c.imageStore = ctdmetadata.NewImageStore(metadataDB)
	c.contentStore = containerdsnapshot.NewContentStore(metadataDB.ContentStore(), "buildkit")

	return nil
}
