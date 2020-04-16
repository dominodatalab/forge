package client

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/containerd/containerd/errdefs"
	"github.com/containerd/containerd/images"
	"github.com/docker/distribution/reference"
)

func (c *Client) TagImage(ctx context.Context, src, dest string) error {
	// Parse the image name and tag for the src image.
	named, err := reference.ParseNormalizedNamed(src)
	if err != nil {
		return fmt.Errorf("parsing image name %q failed: %v", src, err)
	}
	// Add the latest lag if they did not provide one.
	named = reference.TagNameOnly(named)
	src = named.String()

	// Parse the image name and tag for the dest image.
	named, err = reference.ParseNormalizedNamed(dest)
	if err != nil {
		return fmt.Errorf("parsing image name %q failed: %v", dest, err)
	}
	// Add the latest lag if they did not provide one.
	named = reference.TagNameOnly(named)
	dest = named.String()

	if c.workerOpt == nil {
		// Create the worker opts.
		if _, err := c.createWorkerOpt(true); err != nil {
			return fmt.Errorf("creating worker opt failed: %v", err)
		}
	}

	if c.workerOpt.ImageStore == nil {
		return errors.New("image store is nil")
	}

	// Get the source image.
	image, err := c.workerOpt.ImageStore.Get(ctx, src)
	if err != nil {
		return fmt.Errorf("getting image %s from image store failed: %v", src, err)
	}

	// Update the target image. Create it if it does not exist.
	img := images.Image{
		Name:      dest,
		Target:    image.Target,
		CreatedAt: time.Now(),
	}
	if _, err := c.workerOpt.ImageStore.Update(ctx, img); err != nil {
		if !errdefs.IsNotFound(err) {
			return fmt.Errorf("updating image store for %s failed: %v", dest, err)
		}

		// Create it if we didn't find it.
		if _, err := c.workerOpt.ImageStore.Create(ctx, img); err != nil {
			return fmt.Errorf("creating image in image store for %s failed: %v", dest, err)
		}
	}

	return nil
}
