package bkimage

import (
	"context"
	"fmt"

	"github.com/moby/buildkit/util/push"
	log "github.com/sirupsen/logrus"
)

func (c *Client) PushImage(ctx context.Context, image string) error {
	image, err := parseImageName(image)
	if err != nil {
		return err
	}

	// grab reference to local image object
	imgObj, err := c.imageStore.Get(ctx, image)
	if err != nil {
		return fmt.Errorf("getting image %q from image store failed: %w", image, err)
	}

	sm, err := c.getSessionManager()
	if err != nil {
		return err
	}

	log.Infof("Pushing image %q", image)

	// push with context absent session to avoid authorizer override
	// see github.com/moby/buildkit@v0.7.1/util/resolver/resolver.go:158 for more details
	ctx = context.Background()

	// "insecure" param is not used in the following call
	insecure := false
	return push.Push(ctx, sm, c.contentStore, imgObj.Target.Digest, image, insecure, c.getRegistryHosts(), false)
}
