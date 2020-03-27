package runc

//import (
//	"context"
//	"errors"
//	"fmt"
//	"os"
//	"strconv"
//	"strings"
//	"time"
//
//	"github.com/containerd/console"
//	"github.com/moby/buildkit/client"
//	"github.com/moby/buildkit/cmd/buildctl/build"
//	"github.com/moby/buildkit/session"
//	"github.com/moby/buildkit/session/auth/authprovider"
//	"github.com/moby/buildkit/util/progress/progressui"
//	"golang.org/x/sync/errgroup"
//
//	"github.com/dominodatalab/forge/pkg/archive"
//	"github.com/dominodatalab/forge/pkg/container/config"
//)
//
//const defaultTimeout = 300 * time.Second
//
//type builder struct {
//	bk        *client.Client
//	extractor archive.Extractor
//}
//
//func NewImageBuilder() *builder {
//	return &builder{extractor: archive.FetchAndExtract}
//}
//
//func (b *builder) Init() error {
//	bkURL, err := EnsureBuildkitDaemon()
//	if err != nil {
//		return fmt.Errorf("failed to deploy buildkitd: %w", err)
//	}
//
//	bkClient, err := client.New(context.Background(), bkURL)
//	if err != nil {
//		return fmt.Errorf("cannot create buildkit client: %w", err)
//	}
//	b.bk = bkClient
//
//	return nil
//}
//
//func (b *builder) Build(ctx context.Context, opts config.BuildOptions) (string, error) {
//	if b.bk == nil {
//		return "", errors.New("you must invoke Init() before Build()")
//	}
//
//	solveopt, err := b.prepareSolveOpt(opts)
//	if err != nil {
//		return "", err
//	}
//	imageURL := solveopt.Exports[0].Attrs["name"]
//
//	cff, err := console.ConsoleFromFile(os.Stderr)
//	if err != nil {
//		return "", err
//	}
//
//	ch := make(chan *client.SolveStatus)
//
//	if opts.Timeout == 0 {
//		opts.Timeout = defaultTimeout
//	}
//	ctx, cancel := context.WithTimeout(ctx, opts.Timeout)
//	defer cancel()
//
//	eg, _ := errgroup.WithContext(ctx)
//	eg.Go(func() error {
//		var digest string
//
//		resp, err := b.bk.Solve(ctx, nil, *solveopt, ch)
//		if err != nil {
//			return err
//		}
//
//		for k, v := range resp.ExporterResponse {
//			if k == "containerimage.digest" {
//				digest = v
//			}
//		}
//
//		if !strings.ContainsAny(opts.ImageName, ":@") {
//			imageURL = fmt.Sprintf("%s@%s", imageURL, digest)
//		}
//		return nil
//	})
//
//	eg.Go(func() error {
//		return progressui.DisplaySolveStatus(ctx, "", cff, os.Stdout, ch)
//	})
//
//	if err := eg.Wait(); err != nil {
//		return "", err
//	}
//	return imageURL, nil
//}
//
//func (b *builder) prepareSolveOpt(opts config.BuildOptions) (*client.SolveOpt, error) {
//	localCtx, err := b.extractor(opts.Context)
//	if err != nil {
//		return nil, err
//	}
//
//	solveOpt := &client.SolveOpt{
//		Frontend: "dockerfile.v0",
//		FrontendAttrs: map[string]string{
//			"filename": "Dockerfile",
//		},
//		LocalDirs: map[string]string{
//			"context":    localCtx.ContentsDir,
//			"dockerfile": localCtx.ContentsDir,
//		},
//		Session: []session.Attachable{authprovider.NewDockerAuthProvider(os.Stderr)},
//		Exports: []client.ExportEntry{
//			{
//				Type: "image",
//				Attrs: map[string]string{
//					"name":              fmt.Sprintf("%s/%s", opts.RegistryURL, opts.ImageName),
//					"push":              "true",
//					"registry.insecure": strconv.FormatBool(opts.InsecureRegistry),
//				},
//			},
//		},
//	}
//
//	if opts.NoCache {
//		solveOpt.FrontendAttrs["no-cache"] = ""
//	}
//
//	if len(opts.BuildArgs) != 0 {
//		var buildArgs []string
//		for _, arg := range opts.BuildArgs {
//			buildArgs = append(buildArgs, fmt.Sprintf("build-arg:%s", arg))
//		}
//
//		attrsArgs, err := build.ParseOpt(buildArgs, nil)
//		if err != nil {
//			return nil, err
//		}
//		for k, v := range attrsArgs {
//			solveOpt.FrontendAttrs[k] = v
//		}
//	}
//
//	for k, v := range opts.Labels {
//		solveOpt.FrontendAttrs[fmt.Sprintf("label:%s", k)] = v
//	}
//
//	return solveOpt, nil
//}
