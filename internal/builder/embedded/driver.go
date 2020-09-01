package embedded

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/containerd/containerd/namespaces"
	"github.com/docker/distribution/reference"
	"github.com/go-logr/logr"
	controlapi "github.com/moby/buildkit/api/services/control"
	"github.com/moby/buildkit/identity"
	"github.com/moby/buildkit/session"
	"github.com/pkg/errors"
	"golang.org/x/sync/errgroup"

	"github.com/dominodatalab/forge/internal/archive"
	"github.com/dominodatalab/forge/internal/builder/embedded/bkimage"
	"github.com/dominodatalab/forge/internal/builder/embedded/bkimage/types"
	"github.com/dominodatalab/forge/internal/config"
	"github.com/dominodatalab/forge/plugins/preparer"
)

type driver struct {
	bk               *bkimage.Client
	logger           logr.Logger
	preparerPlugins  []*preparer.Plugin
	contextExtractor archive.Extractor
	cacheImageLayers bool
}

func NewDriver(preparerPlugins []*preparer.Plugin, cacheImageLayers bool, logger logr.Logger) (*driver, error) {
	client, err := bkimage.NewClient(config.GetStateDir(), types.AutoBackend, logger)
	if err != nil {
		return nil, errors.Wrap(err, "cannot create buildkit client")
	}

	return &driver{
		bk:               client,
		logger:           logger,
		preparerPlugins:  preparerPlugins,
		contextExtractor: archive.FetchAndExtract,
		cacheImageLayers: cacheImageLayers,
	}, nil
}

func (d *driver) SetLogger(logger logr.Logger) {
	d.logger = logger
	d.bk.SetLogger(logger)
}

func (d *driver) BuildAndPush(ctx context.Context, opts *config.BuildOptions) ([]string, error) {
	if len(opts.PushRegistries) == 0 {
		return nil, errors.New("image builds require at least one push registry")
	}

	// configure registry hosts for every run and reset afterwards
	d.bk.ConfigureHosts(generateRegistryFunc(opts.Registries))
	defer func() { d.bk.ResetHostConfigurations() }()

	var headImg string
	var images []string
	for idx, registry := range opts.PushRegistries {
		// Build fully-qualified image name
		image := fmt.Sprintf("%s/%s", registry, opts.ImageName)

		// Parse the image name and tag.
		named, err := reference.ParseNormalizedNamed(image)
		if err != nil {
			return nil, fmt.Errorf("parsing image name %q failed: %v", image, err)
		}

		// Add the latest tag if they did not provide one.
		named = reference.TagNameOnly(named)
		image = named.String()

		if idx == 0 { // Build, check image size, and set ref to head image
			headImg = image

			if err := d.build(ctx, headImg, opts); err != nil {
				return nil, err
			}
			if opts.ImageSizeLimit != 0 {
				if err := d.validateImageSize(ctx, headImg, opts.ImageSizeLimit); err != nil {
					return nil, err
				}
			}
		} else { // Tag tail images
			if err := d.tag(ctx, headImg, image); err != nil {
				return nil, err
			}
		}

		// Push image into registry
		if err := d.push(ctx, image); err != nil {
			return nil, err
		}
		images = append(images, image)
	}

	// Return a list of every registry image
	return images, nil
}

func (d *driver) build(ctx context.Context, image string, opts *config.BuildOptions) error {
	// download and extract remote OCI context
	extract, err := archive.FetchAndExtract(opts.ContextURL)
	if err != nil {
		return err
	}
	defer os.RemoveAll(extract.RootDir)

	for _, preparerPlugin := range d.preparerPlugins {
		defer func() {
			if err := preparerPlugin.Cleanup(); err != nil {
				d.logger.Error(err, "Error cleaning up prepared resources")
			}
		}()

		d.logger.Info("Preparing resources for image build context")
		if err := preparerPlugin.Prepare(extract.ContentsDir, opts.PluginData); err != nil {
			return err
		}
		d.logger.Info("Resource preparation complete")
		d.logger.Info(strings.Repeat("=", 70))
	}

	// assume Dockerfile lives inside context root
	localDirs := map[string]string{
		"context":    extract.ContentsDir,
		"dockerfile": extract.ContentsDir,
	}

	// create a new buildkit session
	sess, sessDialer, err := d.bk.Session(ctx, localDirs)
	if err != nil {
		return err
	}

	// prepare build parameters
	solveReq, err := solveRequestWithContext(sess.ID(), image, d.cacheImageLayers, opts)
	if err != nil {
		sess.Close()
		return err
	}

	// add build metadata to context
	ctx = session.NewContext(ctx, sess.ID())
	ctx = namespaces.WithNamespace(ctx, "buildkit")
	eg, ctx := errgroup.WithContext(ctx)

	// launch build
	ch := make(chan *controlapi.StatusResponse)
	eg.Go(func() error {
		return sess.Run(ctx, sessDialer)
	})
	eg.Go(func() error {
		defer sess.Close()
		return d.bk.Solve(ctx, solveReq, ch)
	})
	eg.Go(func() error { return displayProgress(ch, &bkimage.LogrWriter{Logger: d.logger}) })

	// return error when one occurs
	return eg.Wait()
}

func (d *driver) tag(ctx context.Context, image, target string) error {
	id := identity.NewID()
	ctx = session.NewContext(ctx, id)
	ctx = namespaces.WithNamespace(ctx, "buildkit")

	return d.bk.TagImage(ctx, image, target)
}

func (d *driver) push(ctx context.Context, image string) error {
	sess, sessDialer, err := d.bk.Session(ctx, nil)
	if err != nil {
		return err
	}

	ctx = session.NewContext(ctx, sess.ID())
	ctx = namespaces.WithNamespace(ctx, "buildkit")
	eg, ctx := errgroup.WithContext(ctx)

	eg.Go(func() error {
		return sess.Run(ctx, sessDialer)
	})
	eg.Go(func() error {
		defer sess.Close()
		return d.bk.PushImage(ctx, image)
	})
	if err := eg.Wait(); err != nil {
		return err
	}

	return nil
}

func (d *driver) validateImageSize(ctx context.Context, name string, limit uint64) error {
	id := identity.NewID()
	ctx = session.NewContext(ctx, id)
	ctx = namespaces.WithNamespace(ctx, "buildkit")

	image, err := d.bk.GetImage(ctx, name)
	if err != nil {
		return fmt.Errorf("cannot validate image size: %v", err)
	}

	imageSize := uint64(image.ContentSize)
	if imageSize > limit {
		return fmt.Errorf("image %q is too large to push to registry (size: %d, limit: %d)", name, imageSize, limit)
	}

	return nil
}
