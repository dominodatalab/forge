package embedded

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/containerd/containerd/namespaces"
	"github.com/docker/distribution/reference"
	"github.com/go-logr/logr"
	controlapi "github.com/moby/buildkit/api/services/control"
	"github.com/moby/buildkit/identity"
	"github.com/moby/buildkit/session"
	"golang.org/x/sync/errgroup"

	"github.com/dominodatalab/forge/internal/archive"
	"github.com/dominodatalab/forge/internal/builder/config"
	"github.com/dominodatalab/forge/internal/builder/embedded/bkimage"
	"github.com/dominodatalab/forge/internal/builder/embedded/bkimage/types"
)

type driver struct {
	bk               *bkimage.Client
	logger           logr.Logger
	contextExtractor archive.Extractor
}

func NewDriver(logger logr.Logger) (*driver, error) {
	client, err := bkimage.NewClient(getStateDir(), types.AutoBackend, logger)
	if err != nil {
		return nil, fmt.Errorf("cannot create buildkit client: %w", err)
	}

	return &driver{
		bk:               client,
		logger:           logger,
		contextExtractor: archive.FetchAndExtract,
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
	defer d.bk.ResetHostConfigurations()

	var images []string
	for _, registry := range opts.PushRegistries {
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

		images = append(images, image)
	}

	if err := d.build(ctx, images, opts); err != nil {
		return nil, err
	}

	if opts.ImageSizeLimit != 0 {
		//noinspection GoNilness
		if err := d.validateImageSize(ctx, images[0], opts.ImageSizeLimit); err != nil {
			return nil, err
		}
	}

	// Return a list of every registry image
	return images, nil
}

func (d *driver) build(ctx context.Context, images []string, opts *config.BuildOptions) error {
	// download and extract remote OCI context
	extract, err := archive.FetchAndExtract(opts.ContextURL)
	if err != nil {
		return err
	}
	defer os.RemoveAll(extract.RootDir)

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
	solveReq, err := solveRequestWithContext(sess.ID(), images, opts)
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
