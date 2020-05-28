package bkimage

import (
	"context"
	"fmt"

	"github.com/containerd/containerd/errdefs"
	"github.com/containerd/containerd/images"
)

func (c *Client) TagImage(ctx context.Context, src, dest string) error {
	src, err := parseImageName(src)
	if err != nil {
		return err
	}

	dest, err = parseImageName(dest)
	if err != nil {
		return err
	}

	// grab reference to local image object
	imgObj, err := c.imageStore.Get(ctx, src)
	if err != nil {
		return fmt.Errorf("getting image %q from image store failed: %w", src, err)
	}

	img := images.Image{
		Name:      dest,
		Target:    imgObj.Target,
		CreatedAt: imgObj.CreatedAt,
	}
	if _, err := c.imageStore.Update(ctx, img); err != nil {
		if !errdefs.IsNotFound(err) {
			return fmt.Errorf("updating image store with %q failed: %w", dest, err)
		}

		if _, err := c.imageStore.Create(ctx, img); err != nil {
			return fmt.Errorf("creating image %q in image store failed: %w", dest, err)
		}
	}

	return nil
}
