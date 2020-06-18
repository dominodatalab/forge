package bkimage

import (
	"context"

	"github.com/containerd/containerd/images"
	"github.com/containerd/containerd/platforms"
	"github.com/pkg/errors"
)

type ListedImage struct {
	images.Image
	ContentSize int64
}

func (c *Client) GetImage(ctx context.Context, name string) (*ListedImage, error) {
	name, err := parseImageName(name)
	if err != nil {
		return nil, err
	}

	imgObj, err := c.imageStore.Get(ctx, name)
	if err != nil {
		return nil, errors.Wrapf(err, "getting image %q from image store failed", name)
	}

	size, err := imgObj.Size(ctx, c.contentStore, platforms.Default())
	if err != nil {
		return nil, errors.Wrapf(err, "calculating image size of %q failed", name)
	}

	return &ListedImage{
		Image:       imgObj,
		ContentSize: size,
	}, nil
}
