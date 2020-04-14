package client

import (
	"context"
	"fmt"

	"github.com/containerd/containerd/remotes/docker"
	"github.com/docker/distribution/reference"
	"github.com/moby/buildkit/util/push"
	"github.com/moby/buildkit/util/resolver"
	"github.com/moby/buildkit/util/tracing"
)

// Push sends an image to a remote registry.
func (c *Client) Push(ctx context.Context, image string, insecure bool, username, password string) error {
	// Parse the image name and tag.
	named, err := reference.ParseNormalizedNamed(image)
	if err != nil {
		return fmt.Errorf("parsing image name %q failed: %v", image, err)
	}
	// Add the latest lag if they did not provide one.
	named = reference.TagNameOnly(named)
	image = named.String()

	if c.workerOpt == nil {
		// Create the worker opts.
		if _, err := c.createWorkerOpt(true); err != nil {
			return fmt.Errorf("creating worker opt failed: %v", err)
		}
	}

	var resolveFn resolver.ResolveOptionsFunc
	if username != "" && password != "" {
		credentials := func(string) (string, string, error) {
			return username, password, nil
		}

		resolveFn = func(ref string) docker.ResolverOptions {
			// NOTE: take from NewResolveOptionsFunc (github.com/moby/buildkit@v0.4.0/util/resolver/resolver.go)
			return docker.ResolverOptions{
				Authorizer: docker.NewAuthorizer(nil, credentials),
				Client:     tracing.DefaultClient,
			}
		}
	} else {
		resolveFn = c.workerOpt.ResolveOptionsFunc
	}

	imgObj, err := c.workerOpt.ImageStore.Get(ctx, image)
	if err != nil {
		return fmt.Errorf("getting image %q failed: %v", image, err)
	}

	sm, err := c.getSessionManager()
	if err != nil {
		return err
	}
	return push.Push(ctx, sm, c.workerOpt.ContentStore, imgObj.Target.Digest, image, insecure, resolveFn)
}
