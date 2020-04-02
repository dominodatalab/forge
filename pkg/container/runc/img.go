package runc

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/containerd/console"
	"github.com/containerd/containerd/namespaces"
	controlapi "github.com/moby/buildkit/api/services/control"
	bkclient "github.com/moby/buildkit/client"
	"github.com/moby/buildkit/cmd/buildctl/build"
	"github.com/moby/buildkit/identity"
	"github.com/moby/buildkit/session"
	"github.com/moby/buildkit/util/progress/progressui"
	"golang.org/x/sync/errgroup"

	"github.com/dominodatalab/forge/pkg/archive"
	"github.com/dominodatalab/forge/pkg/container/config"
	imgclient "github.com/dominodatalab/forge/pkg/img/client"
	"github.com/dominodatalab/forge/pkg/img/types"
)

type Builder struct {
	client *imgclient.Client
}

func NewImgBuilder() (*Builder, error) {
	c, err := imgclient.New(getStateDirectory(), types.AutoBackend)
	if err != nil {
		return nil, err
	}

	return &Builder{client: c}, nil
}

func (b *Builder) Build(ctx context.Context, opts config.BuildOptions) (string, error) {
	name, err := b.build(ctx, opts)
	if err != nil {
		return "", err
	}

	if opts.SizeLimit != 0 {
		if err := b.validateImageSize(ctx, name, opts.SizeLimit); err != nil {
			return "", err
		}
	}

	return name, nil
}

func (b *Builder) build(ctx context.Context, opts config.BuildOptions) (string, error) {
	// download and extract remote OCI context
	extract, err := archive.FetchAndExtract(opts.Context)
	if err != nil {
		return "", err
	}
	defer os.RemoveAll(extract.RootDir)

	// assume Dockerfile lives inside context root
	localDirs := map[string]string{
		"context":    extract.ContentsDir,
		"dockerfile": extract.ContentsDir,
	}

	// create a new buildkit session
	sess, sessDialer, err := b.client.Session(ctx, localDirs)
	if err != nil {
		return "", err
	}

	// prepare build parameters
	solveReq, err := solveRequestWithContext(sess.ID(), opts)
	if err != nil {
		return "", err
	}

	// add build metadata to context
	ctx = session.NewContext(ctx, sess.ID())
	ctx = namespaces.WithNamespace(ctx, "buildkit")
	eg, ctx := errgroup.WithContext(ctx)

	// launch build
	ch := make(chan *controlapi.StatusResponse)
	eg.Go(func() error {
		return sess.Run(ctx, sessDialer)
	})
	eg.Go(func() error {
		defer sess.Close()
		return b.client.Solve(ctx, solveReq, ch)
	})
	eg.Go(func() error {
		return showProgress(ch, false)
	})
	if err := eg.Wait(); err != nil {
		return "", err
	}

	// return final image url
	return solveReq.ExporterAttrs["name"], nil
}

func (b *Builder) validateImageSize(ctx context.Context, name string, limit uint64) error {
	id := identity.NewID()
	ctx = session.NewContext(ctx, id)
	ctx = namespaces.WithNamespace(ctx, "buildkit")

	images, err := b.client.ListImages(ctx, fmt.Sprintf("name==%s", name))
	if err != nil {
		return err
	}
	if len(images) != 1 {
		return fmt.Errorf("could not find exact image %q in list: %v", name, images)
	}

	imageSize := uint64(images[0].ContentSize)
	if imageSize > limit {
		return fmt.Errorf("image %q is too large to push to registry (size: %d, limit: %d)", name, imageSize, limit)
	}

	return nil
}

func getStateDirectory() string {
	xdgDataHome := os.Getenv("XDG_DATA_HOME")
	if xdgDataHome != "" {
		dirs := strings.Split(xdgDataHome, ":")
		return filepath.Join(dirs[0], "forge")
	}
	home := os.Getenv("HOME")
	if home != "" {
		return filepath.Join(home, ".local", "share", "forge")
	}
	return "/tmp/forge"
}

func solveRequestWithContext(sessionID string, opts config.BuildOptions) (*controlapi.SolveRequest, error) {
	req := &controlapi.SolveRequest{
		Ref:      identity.NewID(),
		Session:  sessionID,
		Frontend: "dockerfile.v0",
		FrontendAttrs: map[string]string{
			"filename": "Dockerfile",
		},
		Exporter: "image",
		ExporterAttrs: map[string]string{
			"name": fmt.Sprintf("%s/%s", opts.RegistryURL, opts.ImageName),
		},
	}

	if opts.NoCache {
		req.FrontendAttrs["no-cache"] = ""
	}

	if len(opts.BuildArgs) != 0 {
		var buildArgs []string
		for _, arg := range opts.BuildArgs {
			buildArgs = append(buildArgs, fmt.Sprintf("build-arg:%s", arg))
		}

		attrsArgs, err := build.ParseOpt(buildArgs, nil)
		if err != nil {
			return nil, err
		}
		for k, v := range attrsArgs {
			req.FrontendAttrs[k] = v
		}
	}

	for k, v := range opts.Labels {
		req.FrontendAttrs[fmt.Sprintf("label:%s", k)] = v
	}

	return req, nil
}

func showProgress(ch chan *controlapi.StatusResponse, noConsole bool) error {
	displayCh := make(chan *bkclient.SolveStatus)
	go func() {
		for resp := range ch {
			s := bkclient.SolveStatus{}
			for _, v := range resp.Vertexes {
				s.Vertexes = append(s.Vertexes, &bkclient.Vertex{
					Digest:    v.Digest,
					Inputs:    v.Inputs,
					Name:      v.Name,
					Started:   v.Started,
					Completed: v.Completed,
					Error:     v.Error,
					Cached:    v.Cached,
				})
			}
			for _, v := range resp.Statuses {
				s.Statuses = append(s.Statuses, &bkclient.VertexStatus{
					ID:        v.ID,
					Vertex:    v.Vertex,
					Name:      v.Name,
					Total:     v.Total,
					Current:   v.Current,
					Timestamp: v.Timestamp,
					Started:   v.Started,
					Completed: v.Completed,
				})
			}
			for _, v := range resp.Logs {
				s.Logs = append(s.Logs, &bkclient.VertexLog{
					Vertex:    v.Vertex,
					Stream:    int(v.Stream),
					Data:      v.Msg,
					Timestamp: v.Timestamp,
				})
			}
			displayCh <- &s
		}
		close(displayCh)
	}()
	var c console.Console
	if !noConsole {
		if cf, err := console.ConsoleFromFile(os.Stderr); err == nil {
			c = cf
		}
	}
	return progressui.DisplaySolveStatus(context.TODO(), "", c, os.Stdout, displayCh)
}
