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

	// create worker opt if missing
	if c.workerOpt == nil { // NOTE: modified
		opt, err := c.createWorkerOpt()
		if err != nil {
			return nil, fmt.Errorf("created worker opt failed: %w", err)
		}
		c.workerOpt = &opt
	}

	imgObj, err := c.workerOpt.ImageStore.Get(ctx, name)
	if err != nil {
		return nil, fmt.Errorf("getting image %q from image store failed: %w", name, err)
	}

	size, err := imgObj.Size(ctx, c.workerOpt.ContentStore, platforms.Default())
	if err != nil {
		return nil, fmt.Errorf("calculating image size of %q failed: %w", name, err)
	}

	return &ListedImage{
		Image:       imgObj,
		ContentSize: size,
	}, nil
}
