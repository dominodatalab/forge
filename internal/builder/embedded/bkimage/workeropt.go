package bkimage

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"syscall"

	"github.com/containerd/containerd/diff/apply"
	"github.com/containerd/containerd/diff/walking"
	ctdmetadata "github.com/containerd/containerd/metadata"
	"github.com/containerd/containerd/platforms"
	bkmetadata "github.com/moby/buildkit/cache/metadata"
	"github.com/moby/buildkit/executor/oci"
	"github.com/moby/buildkit/executor/runcexecutor"
	containerdsnapshot "github.com/moby/buildkit/snapshot/containerd"
	"github.com/moby/buildkit/util/binfmt_misc"
	"github.com/moby/buildkit/util/leaseutil"
	"github.com/moby/buildkit/util/network/netproviders"
	"github.com/moby/buildkit/worker/base"
	specs "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/opencontainers/runc/libcontainer/system"
)

func (c *Client) createWorkerOpt() (opt base.WorkerOpt, err error) {
	md, err := bkmetadata.NewStore(filepath.Join(c.rootDir, "metadata.db"))
	if err != nil {
		return opt, err
	}

	// worker executor
	unprivileged := system.GetParentNSeuid() != 0
	c.logger.V(1).Info(fmt.Sprintf("Executor running unprivileged: %t", unprivileged))

	exeOpt := runcexecutor.Opt{
		Root:        filepath.Join(c.rootDir, "executor"),
		Rootless:    unprivileged,
		ProcessMode: c.getProcessMode(),
	}

	np, err := netproviders.Providers(netproviders.Opt{Mode: "auto"})
	if err != nil {
		return opt, err
	}

	exe, err := runcexecutor.New(exeOpt, np)
	if err != nil {
		return opt, err
	}

	// worker metadata
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
		Snapshotter:     containerdsnapshot.NewSnapshotter(c.backend, c.metadataDB.Snapshotter(c.backend), "buildkit", nil),
		ContentStore:    c.contentStore,
		Applier:         apply.NewFileSystemApplier(c.contentStore),
		Differ:          walking.NewWalkingDiff(c.contentStore),
		ImageStore:      c.imageStore,
		RegistryHosts:   c.getRegistryHosts(),
		IdentityMapping: nil,
		LeaseManager:    leaseutil.WithNamespace(ctdmetadata.NewLeaseManager(c.metadataDB), "buildkit"),
		GarbageCollect:  c.metadataDB.GarbageCollect,
	}
	c.workerOpt = &opt // NOTE: modified

	return
}

func (c *Client) getProcessMode() oci.ProcessMode {
	mountArgs := []string{"-t", "proc", "none", "/proc"}
	cmd := exec.Command("mount", mountArgs...)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Pdeathsig:    syscall.SIGKILL,
		Cloneflags:   syscall.CLONE_NEWPID,
		Unshareflags: syscall.CLONE_NEWNS,
	}

	if bs, err := cmd.CombinedOutput(); err != nil {
		c.logger.V(1).Info(fmt.Sprintf("Process sandbox is not available, consider unmasking procfs: %v", string(bs)))
		return oci.NoProcessSandbox
	}
	return oci.ProcessSandbox
}
