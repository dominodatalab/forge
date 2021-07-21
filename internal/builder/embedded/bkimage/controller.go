package bkimage

import (
	"context"
	"path/filepath"

	"github.com/moby/buildkit/cache/remotecache"
	registryremotecache "github.com/moby/buildkit/cache/remotecache/registry"
	"github.com/moby/buildkit/control"
	"github.com/moby/buildkit/frontend"
	"github.com/moby/buildkit/frontend/dockerfile/builder"
	"github.com/moby/buildkit/frontend/gateway"
	"github.com/moby/buildkit/frontend/gateway/forwarder"
	"github.com/moby/buildkit/solver/bboltcachestorage"
	"github.com/moby/buildkit/worker"
	"github.com/moby/buildkit/worker/base"
	"github.com/pkg/errors"
)

func (c *Client) createController(ctx context.Context) error {
	// grab the session manager
	sm, err := c.getSessionManager()
	if err != nil {
		return errors.Wrap(err, "creating session manager failed")
	}

	// create the worker opts
	opt, err := c.createWorkerOpt()
	if err != nil {
		return errors.Wrap(err, "creating worker opt failed")
	}

	// create a new worker
	w, err := base.NewWorker(ctx, opt)
	if err != nil {
		return errors.Wrap(err, "creating worker failed")
	}

	// create a worker controller and add the worker
	wc := &worker.Controller{}
	if err := wc.Add(w); err != nil {
		return errors.Wrap(err, "adding worker to worker controller failed")
	}

	// create the cache store
	cacheStore, err := bboltcachestorage.NewStore(filepath.Join(c.rootDir, "cache.db"))
	if err != nil {
		return errors.Wrap(err, "creating cache store failed")
	}

	// create the controller
	frontends := map[string]frontend.Frontend{
		"dockerfile.v0": forwarder.NewGatewayForwarder(wc, builder.Build),
		"gateway.v0":    gateway.NewGatewayFrontend(wc),
	}
	remoteCacheExporterFuncs := map[string]remotecache.ResolveCacheExporterFunc{
		"registry": registryremotecache.ResolveCacheExporterFunc(sm, c.getRegistryHosts()),
	}
	remoteCacheImporterFuncs := map[string]remotecache.ResolveCacheImporterFunc{
		"registry": registryremotecache.ResolveCacheImporterFunc(sm, opt.ContentStore, c.getRegistryHosts()),
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
		return errors.Wrap(err, "creating controller failed")
	}

	c.controller = controller
	return nil
}
