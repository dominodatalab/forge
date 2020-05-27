package bkimage

import (
	"fmt"
	"path/filepath"

	"github.com/containerd/containerd/remotes/docker"
	"github.com/moby/buildkit/cache/remotecache"
	inlineremotecache "github.com/moby/buildkit/cache/remotecache/inline"
	registryremotecache "github.com/moby/buildkit/cache/remotecache/registry"
	"github.com/moby/buildkit/control"
	"github.com/moby/buildkit/frontend"
	"github.com/moby/buildkit/frontend/dockerfile/builder"
	"github.com/moby/buildkit/frontend/gateway"
	"github.com/moby/buildkit/frontend/gateway/forwarder"
	"github.com/moby/buildkit/solver/bboltcachestorage"
	"github.com/moby/buildkit/worker"
	"github.com/moby/buildkit/worker/base"
)

func (c *Client) createController() error {
	// grab the session manager
	sm, err := c.getSessionManager()
	if err != nil {
		return fmt.Errorf("creating session manager failed: %w", err)
	}

	// create the worker opts
	opt, err := c.createWorkerOpt(true)
	if err != nil {
		return fmt.Errorf("creating worker opt failed: %w", err)
	}

	// create a new worker
	w, err := base.NewWorker(opt)
	if err != nil {
		return fmt.Errorf("creating worker failed: %w", err)
	}

	// create a worker controller and add the worker
	wc := &worker.Controller{}
	if err := wc.Add(w); err != nil {
		return fmt.Errorf("adding worker to worker controller failed: %w", err)
	}

	// create the cache store
	cacheStore, err := bboltcachestorage.NewStore(filepath.Join(c.rootDir, "cache.db"))
	if err != nil {
		return fmt.Errorf("creating cache store failed: %w", err)
	}

	// create the controller
	frontends := map[string]frontend.Frontend{
		"dockerfile.v0": forwarder.NewGatewayForwarder(wc, builder.Build),
		"gateway.v0":    gateway.NewGatewayFrontend(wc),
	}
	remoteCacheExporterFuncs := map[string]remotecache.ResolveCacheExporterFunc{
		"inline": inlineremotecache.ResolveCacheExporterFunc(),
	}
	remoteCacheImporterFuncs := map[string]remotecache.ResolveCacheImporterFunc{
		"registry": registryremotecache.ResolveCacheImporterFunc(sm, opt.ContentStore, docker.ConfigureDefaultRegistries()),
	}
	controller, err := control.NewController(control.Opt{
		SessionManager:            sm,
		WorkerController:          wc,
		Frontends:                 frontends,
		CacheKeyStorage:           cacheStore,
		ResolveCacheExporterFuncs: remoteCacheExporterFuncs,
		ResolveCacheImporterFuncs: remoteCacheImporterFuncs,
		Entitlements:              nil,
	})
	if err != nil {
		return fmt.Errorf("creating controller failed: %w", err)
	}

	c.controller = controller
	return nil
}
