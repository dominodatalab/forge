package bkimage

import (
	"context"
	"fmt"

	"github.com/containerd/containerd/images"
	"github.com/containerd/containerd/platforms"
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
		return nil, fmt.Errorf("getting image %q from image store failed: %w", name, err)
	}

	size, err := imgObj.Size(ctx, c.contentStore, platforms.Default())
	if err != nil {
		return nil, fmt.Errorf("calculating image size of %q failed: %w", name, err)
	}

	return &ListedImage{
		Image:       imgObj,
		ContentSize: size,
	}, nil
}
