package bkimage

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
	"github.com/moby/buildkit/util/progress"
	"github.com/moby/buildkit/util/push"
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

	// TODO figure out if / how buildkit & containerd convey progress during the push
	c.logger.Info(fmt.Sprintf("Pushing image %q", image))
	defer c.logger.Info(fmt.Sprintf("Pushed image %q", image))

	progressWriter := progress.NewMultiWriter()
	progressWriter.Add(&logrProgressWriter{c.logger})

	// push with context absent session to avoid authorizer override
	// see github.com/moby/buildkit@v0.7.1/util/resolver/resolver.go:158 for more details
	ctx = progress.WithProgress(context.Background(), progressWriter)

	// "insecure" param is not used in the following call
	insecure := false
	return push.Push(ctx, sm, c.contentStore, imgObj.Target.Digest, image, insecure, c.getRegistryHosts(), false)
}

type logrProgressWriter struct {
	logger logr.Logger
}

func (l *logrProgressWriter) Write(id string, value interface{}) error {
	l.logger.Info(id, "progress", value)
	return nil
}

func (l *logrProgressWriter) WriteRawProgress(progress *progress.Progress) error {
	l.logger.Info(progress.ID, "progress", progress)
	return nil
}

func (l *logrProgressWriter) Close() error {
	return nil
}
