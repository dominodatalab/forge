package runc

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/dominodatalab/forge/pkg/archive"

	"github.com/containerd/console"
	"github.com/moby/buildkit/client"
	"github.com/moby/buildkit/session"
	"github.com/moby/buildkit/session/auth/authprovider"
	"github.com/moby/buildkit/util/progress/progressui"
	"golang.org/x/sync/errgroup"

	"github.com/dominodatalab/forge/pkg/container/config"
)

const defaultTimeout = 300 * time.Second

type Builder struct {
	bk      *client.Client
	timeout time.Duration
}

func NewRuncBuilder(addr string) (*Builder, error) {
	bk, err := client.New(context.Background(), addr)
	if err != nil {
		return nil, err
	}

	return &Builder{
		timeout: defaultTimeout,
		bk:      bk,
	}, nil
}

func (b *Builder) Build(ctx context.Context, opts config.BuildOptions) (string, error) {
	solveopt, image, err := PrepareSolveOpt(opts)
	if err != nil {
		return "", err
	}

	ctx, cancel := context.WithTimeout(ctx, b.timeout)
	defer cancel()

	ch := make(chan *client.SolveStatus)
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
			image = fmt.Sprintf("%s@%s", image, digest)
		}
		return nil
	})

	eg.Go(func() error {
		cff, err := console.ConsoleFromFile(os.Stderr)
		if err != nil {
			return err
		}
		return progressui.DisplaySolveStatus(ctx, "", cff, os.Stdout, ch)
	})

	if err := eg.Wait(); err != nil {
		return "", err
	}
	return image, nil
}

func PrepareSolveOpt(opts config.BuildOptions) (*client.SolveOpt, string, error) {
	_, err := archive.FetchAndExtract(opts.Context)
	if err != nil {
		return nil, "", err
	}

	if opts.Dockerfile == "" {
		opts.Dockerfile = "Dockerfile"
	}

	solveopt := client.SolveOpt{
		Frontend: "dockerfile.v0",
		FrontendAttrs: map[string]string{
			"filename": opts.Dockerfile,
		},
		LocalDirs: map[string]string{
			"context":    opts.Context,
			"dockerfile": opts.Context,
		},
		Session: []session.Attachable{authprovider.NewDockerAuthProvider(os.Stderr)},
	}

	if opts.NoCache {
		solveopt.FrontendAttrs["no-cache"] = ""
	}

	image := fmt.Sprintf("%s/%s", opts.RegistryURL, opts.ImageName)
	solveopt.Exports = []client.ExportEntry{
		{
			Type: "image",
			Attrs: map[string]string{
				"name":              image,
				"push":              "true",
				"registry.insecure": strconv.FormatBool(opts.InsecureRegistry),
			},
		},
	}
	return &solveopt, image, nil
}

//func prepareContextDir(contextURL string) (dir string, err error) {
//
//	//dir, err = ioutil.TempDir("", "forge-build")
//	//if err != nil {
//	//	return
//	//}
//	//defer os.RemoveAll(dir)
//	//
//	//fp := filepath.Join(dir, "context.archive")
//	//if err := util.DownloadFile(fp, contextURL); err != nil {
//	//	return
//	//}
//	//
//	//if err := util.ExtractArchive(fp); err != nil {
//	//	return
//	//}
//	//return
//}
