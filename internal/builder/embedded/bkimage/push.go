package bkimage

import (
	"context"
	"fmt"
	"strings"

	"github.com/moby/buildkit/util/push"
	"github.com/opencontainers/go-digest"
	"github.com/pkg/errors"
)

func (c *Client) PushImage(ctx context.Context, id string, image string) error {
	image, err := parseImageName(image)
	if err != nil {
		return err
	}

	// grab reference to local image object
	imgObj, err := c.imageStore.Get(ctx, image)
	if err != nil {
		return errors.Wrapf(err, "getting image %q from image store failed", image)
	}

	sm, err := c.getSessionManager()
	if err != nil {
		return err
	}

	// TODO figure out if / how buildkit & containerd convey progress during the push
	c.logger.Info(strings.Repeat("=", 70))
	c.logger.Info(fmt.Sprintf("Pushing image %q", image))
	defer c.logger.Info(fmt.Sprintf("Pushed image %q", image))

	annotations := map[digest.Digest]map[string]string{imgObj.Target.Digest: imgObj.Target.Annotations}
	// NOTE: "insecure" param is not used in the following func call
	return push.Push(ctx, sm, id, c.contentStore, c.contentStore, imgObj.Target.Digest, image, false, c.getRegistryHosts(), false, annotations)
}
