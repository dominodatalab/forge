package runc

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/containerd/console"
	"github.com/moby/buildkit/client"
	"github.com/moby/buildkit/cmd/buildctl/build"
	"github.com/moby/buildkit/session"
	"github.com/moby/buildkit/session/auth/authprovider"
	"github.com/moby/buildkit/util/progress/progressui"
	"golang.org/x/sync/errgroup"

	"github.com/dominodatalab/forge/pkg/archive"
	"github.com/dominodatalab/forge/pkg/container/config"
)

const defaultTimeout = 300 * time.Second

type Builder struct {
	bk        *client.Client
	timeout   time.Duration
	extractor archive.Extractor
}

func NewRuncBuilder(addr string) (*Builder, error) {
	bk, err := client.New(context.Background(), addr)
	if err != nil {
		return nil, err
	}

	return &Builder{
		timeout:   defaultTimeout,
		bk:        bk,
		extractor: archive.FetchAndExtract,
	}, nil
}

func (b *Builder) Build(ctx context.Context, opts config.BuildOptions) (string, error) {
	solveopt, err := b.PrepareSolveOpt(opts)
	if err != nil {
		return "", err
	}
	imageURL := solveopt.Exports[0].Attrs["name"]

	cff, err := console.ConsoleFromFile(os.Stderr)
	if err != nil {
		return "", err
	}

	ch := make(chan *client.SolveStatus)

	ctx, cancel := context.WithTimeout(ctx, b.timeout)
	defer cancel()

	eg, _ := errgroup.WithContext(ctx)
	eg.Go(func() error {
		var digest string

		resp, err := b.bk.Solve(ctx, nil, *solveopt, ch)
		if err != nil {
			return err
		}

		for k, v := range resp.ExporterResponse {
			if k == "containerimage.digest" {
				digest = v
			}
		}

		if !strings.ContainsAny(opts.ImageName, ":@") {
			imageURL = fmt.Sprintf("%s@%s", imageURL, digest)
		}
		return nil
	})

	eg.Go(func() error {
		return progressui.DisplaySolveStatus(ctx, "", cff, os.Stdout, ch)
	})

	if err := eg.Wait(); err != nil {
		return "", err
	}
	return imageURL, nil
}

func (b *Builder) PrepareSolveOpt(opts config.BuildOptions) (*client.SolveOpt, error) {
	localCtx, err := b.extractor(opts.Context)
	if err != nil {
		return nil, err
	}

	solveOpt := &client.SolveOpt{
		Frontend: "dockerfile.v0",
		FrontendAttrs: map[string]string{
			"filename": "Dockerfile",
		},
		LocalDirs: map[string]string{
			"context":    localCtx.ContentsDir,
			"dockerfile": localCtx.ContentsDir,
		},
		Session: []session.Attachable{authprovider.NewDockerAuthProvider(os.Stderr)},
		Exports: []client.ExportEntry{
			{
				Type: "image",
				Attrs: map[string]string{
					"name":              fmt.Sprintf("%s/%s", opts.RegistryURL, opts.ImageName),
					"push":              "true",
					"registry.insecure": strconv.FormatBool(opts.InsecureRegistry),
				},
			},
		},
	}

	if opts.NoCache {
		solveOpt.FrontendAttrs["no-cache"] = ""
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
			solveOpt.FrontendAttrs[k] = v
		}
	}

	for k, v := range opts.Labels {
		solveOpt.FrontendAttrs[fmt.Sprintf("label:%s", k)] = v
	}

	return solveOpt, nil
}
