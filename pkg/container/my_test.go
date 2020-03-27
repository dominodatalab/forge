package container

import (
	"context"
	"fmt"
	"github.com/containerd/console"
	"github.com/containerd/containerd/namespaces"
	"github.com/genuinetools/img/client"
	"github.com/genuinetools/img/types"
	controlapi "github.com/moby/buildkit/api/services/control"
	bkclient "github.com/moby/buildkit/client"
	"github.com/moby/buildkit/identity"
	"github.com/moby/buildkit/session"
	"github.com/moby/buildkit/util/appcontext"
	"github.com/moby/buildkit/util/progress/progressui"
	"golang.org/x/sync/errgroup"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestThings(t *testing.T) {
	if err := build(); err != nil {
		t.Errorf("%+v\n", err)
	}
}

func build() error {
	stateDir := defaultStateDirectory()
	backend := types.AutoBackend
	localDirs := getLocalDirs()

	c, err := client.New(stateDir, backend, localDirs)
	if err != nil {
		return err
	}
	defer c.Close()

	ctx := appcontext.Context()
	sess, sessDialer, err := c.Session(ctx)
	if err != nil {
		return err
	}
	id := identity.NewID()
	ctx = session.NewContext(ctx, sess.ID())
	ctx = namespaces.WithNamespace(ctx, "buildkit")
	eg, ctx := errgroup.WithContext(ctx)

	ch := make(chan *controlapi.StatusResponse)
	eg.Go(func() error {
		return sess.Run(ctx, sessDialer)
	})
	// Solve the dockerfile.
	eg.Go(func() error {
		defer sess.Close()
		return c.Solve(ctx, &controlapi.SolveRequest{
			Ref:      id,
			Session:  sess.ID(),
			Exporter: "image",
			ExporterAttrs: map[string]string{
				"name": "my-image",
			},
			Frontend: "dockerfile.v0",
			FrontendAttrs: map[string]string{
				"filename": "Dockerfile",
			},
		}, ch)
	})
	eg.Go(func() error {
		noConsole := false
		return showProgress(ch, noConsole)
	})
	if err := eg.Wait(); err != nil {
		return err
	}
	fmt.Printf("Successfully built %s\n", "latest")

	return nil
}

func defaultStateDirectory() string {
	//  pam_systemd sets XDG_RUNTIME_DIR but not other dirs.
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

func getLocalDirs() map[string]string {
	return map[string]string{
		"context":    "/tmp/my-test-context",
		"dockerfile": "/tmp/my-test-context",
	}
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
