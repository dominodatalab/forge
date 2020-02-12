package runc

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/containerd/console"
	"github.com/moby/buildkit/client"
	"github.com/moby/buildkit/session"
	"github.com/moby/buildkit/session/auth/authprovider"
	"github.com/moby/buildkit/util/progress/progressui"
	"golang.org/x/sync/errgroup"

	"github.com/dominodatalab/forge/api/v1alpha1"
)

type Builder struct {
	Timeout        time.Duration
	buildkitClient *client.Client
}

func NewRuncBuilder(host string, port int) (*Builder, error) {
	bkClient, err := client.New(context.TODO(), fmt.Sprintf("tcp://%s:%d", host, port))
	if err != nil {
		return nil, err
	}

	builder := Builder{
		Timeout:        300 * time.Second,
		buildkitClient: bkClient,
	}
	return &builder, nil
}

func (b *Builder) Build(ctx context.Context, spec v1alpha1.ContainerImageBuildSpec) (string, error) {
	solveOpt, teardown, err := prepareBuildContext(spec)
	if err != nil {
		return "", err
	}
	defer teardown()

	image := fmt.Sprintf("%s/%s", spec.Build.PushRegistry, spec.Build.ImageName)
	solveOpt.Exports = []client.ExportEntry{
		{
			Type: "image",
			Attrs: map[string]string{
				"name": image,
				//"push": "true", // TODO add this when ready to push
			},
		},
	}

	ctx, cancel := context.WithTimeout(ctx, b.Timeout)
	defer cancel()

	ch := make(chan *client.SolveStatus)
	eg, _ := errgroup.WithContext(ctx)

	eg.Go(func() error {
		var digest string

		resp, err := b.buildkitClient.Solve(ctx, nil, *solveOpt, ch)
		if err != nil {
			return err
		}

		for k, v := range resp.ExporterResponse {
			if k == "containerimage.digest" {
				digest = v
			}
		}

		if !strings.ContainsAny(spec.Build.ImageName, ":@") {
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

func prepareBuildContext(spec v1alpha1.ContainerImageBuildSpec) (*client.SolveOpt, func(), error) {
	contextDir, err := ioutil.TempDir("", "")
	if err != nil {
		return nil, nil, err
	}
	teardown := func() { os.RemoveAll(contextDir) }

	dockerfile := filepath.Join(contextDir, "Dockerfile")
	contents := []byte(strings.Join(spec.Build.Commands, "\n"))
	if err := ioutil.WriteFile(dockerfile, contents, 0644); err != nil {
		teardown()
		return nil, nil, err
	}

	solveOpt := client.SolveOpt{
		Frontend: "dockerfile.v0",
		FrontendAttrs: map[string]string{
			"filename": filepath.Base(dockerfile),
		},
		LocalDirs: map[string]string{
			"context":    contextDir,
			"dockerfile": contextDir,
		},
		Session: []session.Attachable{authprovider.NewDockerAuthProvider(os.Stderr)},
	}
	return &solveOpt, teardown, nil
}
