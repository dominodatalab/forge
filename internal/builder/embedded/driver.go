package embedded

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/containerd/containerd/namespaces"
	"github.com/docker/distribution/reference"
	controlapi "github.com/moby/buildkit/api/services/control"
	bkclient "github.com/moby/buildkit/client"
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
	contextExtractor archive.Extractor
}

func NewDriver() (*driver, error) {
	client, err := bkimage.NewClient(getStateDir(), types.AutoBackend)
	if err != nil {
		return nil, fmt.Errorf("cannot create buildkit client: %w", err)
	}

	return &driver{
		bk:               client,
		contextExtractor: archive.FetchAndExtract,
	}, nil
}

func (d *driver) BuildAndPush(ctx context.Context, opts *config.BuildOptions, progressFunc func(chan *bkclient.SolveStatus) error) ([]string, error) {
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

			if err := d.build(ctx, headImg, opts, progressFunc); err != nil {
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

func (d *driver) build(ctx context.Context, image string, opts *config.BuildOptions, progressFunc func(chan *bkclient.SolveStatus) error) error {
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
	solveReq, err := solveRequestWithContext(sess.ID(), image, opts)
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
	eg.Go(func() error { return displayProgress(ch, progressFunc) })

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
