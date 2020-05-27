package bkimage

import (
	"context"
	"fmt"

	"github.com/containerd/containerd/remotes/docker"
	"github.com/moby/buildkit/util/push"
	log "github.com/sirupsen/logrus"
)

func (c *Client) Push(ctx context.Context, image string, insecure bool, username, password string) error {
	image, err := parseImageName(image)
	if err != nil {
		return err
	}

	// create worker opt if missing
	if c.workerOpt == nil { // NOTE: modified
		opt, err := c.createWorkerOpt()
		if err != nil {
			return fmt.Errorf("created worker opt failed: %w", err)
		}
		c.workerOpt = &opt
	}

	// grab reference to local image object
	imgObj, err := c.workerOpt.ImageStore.Get(ctx, image)
	if err != nil {
		return fmt.Errorf("getting image %q from image store failed: %w", image, err)
	}

	sm, err := c.getSessionManager()
	if err != nil {
		return err
	}

	var rOpts []docker.RegistryOpt
	if insecure {
		opt := docker.WithPlainHTTP(func(s string) (bool, error) {
			return true, nil
		})
		rOpts = append(rOpts, opt)
	}
	if username != "" && password != "" {
		authOpt := docker.WithAuthCreds(func(s string) (string, string, error) {
			return username, password, nil
		})
		authorizer := docker.NewDockerAuthorizer(authOpt)
		opt := docker.WithAuthorizer(authorizer)
		rOpts = append(rOpts, opt)
	}
	hosts := docker.ConfigureDefaultRegistries(rOpts...)

	log.Infof("Pushing image %q", image)

	// push with context absent session to avoid authorizer override
	// see github.com/moby/buildkit@v0.7.1/util/resolver/resolver.go:158 for more details
	ctx = context.Background()

	// "insecure" param is not used in the following call
	return push.Push(ctx, sm, c.workerOpt.ContentStore, imgObj.Target.Digest, image, insecure, hosts, false)
}

// NOTE: trying to figure out how to configure all provided hosts
//func newRegistryConfig(regs []config.Registry) docker.RegistryHosts {
//	regsMap := map[string]config.Registry{}
//	for _, reg := range regs {
//		regsMap[reg.Host] = reg
//	}
//
//	// authentication credentials informer
//	authOpt := docker.WithAuthCreds(func(host string) (string, string, error) {
//		if reg, ok := regsMap[host]; ok {
//			return reg.Username, reg.Password, nil
//		}
//		return "", "", nil
//	})
//	authorizer := docker.NewDockerAuthorizer(authOpt)
//
//	// plain http scheme informer
//	matchProvidedHosts := func(host string) (bool, error) {
//		if reg, ok := regsMap[host]; ok {
//			return reg.NonSSL, nil
//		}
//		return false, nil
//	}
//
//	return docker.ConfigureDefaultRegistries(
//		docker.WithAuthorizer(authorizer),
//		docker.WithPlainHTTP(matchProvidedHosts),
//	)
//}
