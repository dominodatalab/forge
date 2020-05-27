package bkimage

import (
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"syscall"

	"github.com/containerd/containerd/content/local"
	"github.com/containerd/containerd/diff/apply"
	"github.com/containerd/containerd/diff/walking"
	ctdmetadata "github.com/containerd/containerd/metadata"
	"github.com/containerd/containerd/platforms"
	"github.com/containerd/containerd/remotes/docker"
	"github.com/containerd/containerd/snapshots"
	"github.com/containerd/containerd/snapshots/native"
	"github.com/containerd/containerd/snapshots/overlay"
	bkmetadata "github.com/moby/buildkit/cache/metadata"
	"github.com/moby/buildkit/executor"
	"github.com/moby/buildkit/executor/oci"
	"github.com/moby/buildkit/executor/runcexecutor"
	containerdsnapshot "github.com/moby/buildkit/snapshot/containerd"
	"github.com/moby/buildkit/util/binfmt_misc"
	"github.com/moby/buildkit/util/leaseutil"
	"github.com/moby/buildkit/util/network/netproviders"
	"github.com/moby/buildkit/worker/base"
	specs "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/opencontainers/runc/libcontainer/system"
	log "github.com/sirupsen/logrus"
	bolt "go.etcd.io/bbolt"

	"github.com/dominodatalab/forge/internal/builder/embedded/bkimage/types"
)

func (c *Client) createWorkerOpt(withExecutor bool) (opt base.WorkerOpt, err error) {
	md, err := bkmetadata.NewStore(filepath.Join(c.rootDir, "metadata.db"))
	if err != nil {
		return opt, err
	}

	mdb, err := c.getMetadataDB()
	if err != nil {
		return opt, err
	}

	imageStore := ctdmetadata.NewImageStore(mdb)
	contentStore := containerdsnapshot.NewContentStore(mdb.ContentStore(), "buildkit")

	// executor logic
	unprivileged := system.GetParentNSeuid() != 0
	log.Infof("Executor running unprivileged: %t", unprivileged)

	var exe executor.Executor
	if withExecutor {
		exeOpt := runcexecutor.Opt{
			Root:        filepath.Join(c.rootDir, "executor"),
			Rootless:    unprivileged,
			ProcessMode: getProcessMode(),
		}

		np, err := netproviders.Providers(netproviders.Opt{Mode: "auto"})
		if err != nil {
			return opt, err
		}

		exe, err = runcexecutor.New(exeOpt, np)
		if err != nil {
			return opt, err
		}

		fmt.Println(exe)
	}

	// worker opt metadata
	id, err := base.ID(c.rootDir)
	if err != nil {
		return opt, err
	}

	executorLabels := base.Labels("oci", c.backend)

	var supportedPlatforms []specs.Platform
	for _, s := range binfmt_misc.SupportedPlatforms(false) {
		p, err := platforms.Parse(s)
		if err != nil {
			return opt, err
		}
		supportedPlatforms = append(supportedPlatforms, platforms.Normalize(p))
	}

	opt = base.WorkerOpt{
		ID:              id,
		Labels:          executorLabels,
		Platforms:       supportedPlatforms,
		GCPolicy:        nil,
		MetadataStore:   md,
		Executor:        exe,
		Snapshotter:     containerdsnapshot.NewSnapshotter(c.backend, mdb.Snapshotter(c.backend), "buildkit", nil),
		ContentStore:    contentStore,
		Applier:         apply.NewFileSystemApplier(contentStore),
		Differ:          walking.NewWalkingDiff(contentStore),
		ImageStore:      imageStore,
		RegistryHosts:   docker.ConfigureDefaultRegistries(), // TODO: this may be the place to hook in authN for private registries
		IdentityMapping: nil,
		LeaseManager:    leaseutil.WithNamespace(ctdmetadata.NewLeaseManager(mdb), "buildkit"),
		GarbageCollect:  mdb.GarbageCollect,
	}
	c.workerOpt = &opt // NOTE: modified

	return
}

func (c *Client) getMetadataDB() (*ctdmetadata.DB, error) {
	db, err := bolt.Open(filepath.Join(c.rootDir, "container-metadata.db"), 0644, nil)
	if err != nil {
		return nil, err
	}

	cs, err := local.NewStore(filepath.Join(c.rootDir, "content"))
	if err != nil {
		return nil, err
	}

	snapshotDir := filepath.Join(c.rootDir, "snapshotter")
	var snapshotter snapshots.Snapshotter
	switch c.backend {
	case types.NativeBackend:
		snapshotter, err = native.NewSnapshotter(snapshotDir)
	case types.OverlayFSBackend:
		snapshotter, err = overlay.NewSnapshotter(snapshotDir)
	default:
		return nil, fmt.Errorf("%s is not a valid snapshotter", c.backend)
	}
	if err != nil {
		return nil, fmt.Errorf("creating %s snapshotter failed: %w", c.backend, err)
	}

	metadataDB := ctdmetadata.NewDB(db, cs, map[string]snapshots.Snapshotter{
		c.backend: snapshotter,
	})
	if err := metadataDB.Init(context.TODO()); err != nil {
		return nil, err
	}

	return metadataDB, nil
}

func getProcessMode() oci.ProcessMode {
	mountArgs := []string{"-t", "proc", "none", "/proc"}
	cmd := exec.Command("mount", mountArgs...)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Pdeathsig:    syscall.SIGKILL,
		Cloneflags:   syscall.CLONE_NEWPID,
		Unshareflags: syscall.CLONE_NEWNS,
	}

	if bs, err := cmd.CombinedOutput(); err != nil {
		log.Warnf("Process sandbox is not available, consider unmasking procfs: %v", string(bs))
		return oci.NoProcessSandbox
	}
	return oci.ProcessSandbox
}
