package embedded

import (
	"context"
	"fmt"
	"os"

	"github.com/containerd/console"
	controlapi "github.com/moby/buildkit/api/services/control"
	bkclient "github.com/moby/buildkit/client"
	"github.com/moby/buildkit/cmd/buildctl/build"
	"github.com/moby/buildkit/identity"
	"github.com/moby/buildkit/util/progress/progressui"

	"github.com/dominodatalab/forge/internal/builder/embedded/bkimage"
	"github.com/dominodatalab/forge/internal/config"
)

func solveRequestWithContext(sessionID string, image string, opts *config.BuildOptions) (*controlapi.SolveRequest, error) {
	req := &controlapi.SolveRequest{
		Ref:      identity.NewID(),
		Session:  sessionID,
		Frontend: "dockerfile.v0",
		FrontendAttrs: map[string]string{
			"filename": "Dockerfile",
		},
		Exporter: "image",
		ExporterAttrs: map[string]string{
			"name": image,
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

func generateRegistryFunc(registries []config.Registry) (bkimage.CredentialsFn, bkimage.TLSEnabledFn, string) {
	rHostMap := map[string]config.Registry{}
	for _, reg := range registries {
		rHostMap[reg.Host] = reg
	}

	// authentication credentials func
	hostCredentials := func(host string) (string, string, error) {
		if reg, ok := rHostMap[host]; ok {
			return reg.Username, reg.Password, nil
		}
		return "", "", nil
	}

	// plain http scheme func
	matchNonSSL := func(host string) (bool, error) {
		if reg, ok := rHostMap[host]; ok {
			return reg.NonSSL, nil
		}
		return false, nil
	}

	certs := os.Getenv("CERT_DIR")
	return hostCredentials, matchNonSSL, fmt.Sprintf("%s/tls.crt", certs)
}
