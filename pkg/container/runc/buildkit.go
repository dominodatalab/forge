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

type Builder struct{}

func NewRuncBuilder() *Builder {
	return &Builder{}
}

func (b *Builder) Build(ctx context.Context, spec v1alpha1.ContainerImageBuildSpec) (string, error) {
	// buildkit daemon input parameters
	host := "192.168.64.74"
	port := "30138"

	// initialize client
	ctx, cancel := context.WithTimeout(ctx, 300*time.Second)
	defer cancel()

	buildkitClient, err := client.New(ctx, fmt.Sprintf("tcp://%s:%s", host, port))
	if err != nil {
		return "", nil
	}

	contextDir, err := ioutil.TempDir("", "")
	if err != nil {
		return "", err
	}
	defer func() { os.RemoveAll(contextDir) }()

	solveOpt, err := prepareBuildContext(contextDir, spec)
	if err != nil {
		return "", err
	}

	ch := make(chan *client.SolveStatus)
	eg, _ := errgroup.WithContext(ctx)

	eg.Go(func() error {
		_, err := buildkitClient.Solve(ctx, nil, *solveOpt, ch)
		return err
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

	return "", nil
}

func prepareBuildContext(contextDir string, spec v1alpha1.ContainerImageBuildSpec) (*client.SolveOpt, error) {
	dockerfile := filepath.Join(contextDir, "Dockerfile")
	contents := []byte(strings.Join(spec.Build.Commands, "\n"))

	if err := ioutil.WriteFile(dockerfile, contents, 0644); err != nil {
		return nil, err
	}

	attachable := []session.Attachable{authprovider.NewDockerAuthProvider(os.Stderr)}

	return &client.SolveOpt{
		Frontend: "dockerfile.v0",
		FrontendAttrs: map[string]string{
			"filename": filepath.Base(dockerfile),
		},
		LocalDirs: map[string]string{
			"context":    contextDir,
			"dockerfile": contextDir,
		},
		Session: attachable,
		//Exports: []client.ExportEntry{
		//	{
		//		Type: "image",
		//		Attrs: map[string]string{
		//			"name": spec.Build.ImageURL,
		//		},
		//	},
		//},
	}, nil
}
