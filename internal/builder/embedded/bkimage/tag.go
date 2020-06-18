package bkimage

import (
	"context"

	"github.com/containerd/containerd/errdefs"
	"github.com/containerd/containerd/images"
	"github.com/pkg/errors"
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
		return errors.Wrapf(err, "getting image %q from image store failed", src)
	}

	img := images.Image{
		Name:      dest,
		Target:    imgObj.Target,
		CreatedAt: imgObj.CreatedAt,
	}
	if _, err := c.imageStore.Update(ctx, img); err != nil {
		if !errdefs.IsNotFound(err) {
			return errors.Wrapf(err, "updating image store with %q failed", dest)
		}

		if _, err := c.imageStore.Create(ctx, img); err != nil {
			return errors.Wrapf(err, "creating image %q in image store failed", dest)
		}
	}

	return nil
}
