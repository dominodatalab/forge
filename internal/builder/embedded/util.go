package embedded

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/docker/distribution/reference"
	controlapi "github.com/moby/buildkit/api/services/control"
	bkclient "github.com/moby/buildkit/client"
	"github.com/moby/buildkit/cmd/buildctl/build"
	"github.com/moby/buildkit/identity"
	"github.com/moby/buildkit/util/progress/progressui"

	"github.com/dominodatalab/forge/internal/builder/embedded/bkimage"
	"github.com/dominodatalab/forge/internal/config"
)

const (
	// common tag name that will be used when registry image caching is enabled.
	cacheTag = "buildcache"

	// cache mode used when override envvar is not set
	defaultCacheMode = "max"
)

func solveRequestWithContext(sessionID string, image string, cacheImageLayers bool, opts *config.BuildOptions) (*controlapi.SolveRequest, error) {
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
		Cache: controlapi.CacheOptions{},
	}

	imageName, err := reference.ParseNormalizedNamed(image)
	if err != nil {
		return nil, err
	}
	cacheTaggedName, err := reference.WithTag(imageName, cacheTag)
	if err != nil {
		return nil, err
	}
	cacheTagRef := cacheTaggedName.String()

	if cacheImageLayers {
		if !opts.DisableLayerCacheExport {
			cacheMode, err := getExportMode()
			if err != nil {
				return nil, err
			}

			req.Cache.Exports = []*controlapi.CacheOptionsEntry{{
				Type: "registry",
				Attrs: map[string]string{
					"mode": cacheMode,
					"ref":  cacheTagRef,
				},
			}}

			req.FrontendAttrs["cache-from"] = cacheTagRef
		}

		if !opts.DisableBuildCache {
			req.Cache.Imports = []*controlapi.CacheOptionsEntry{{
				Type: "registry",
				Attrs: map[string]string{
					"ref": cacheTagRef,
				},
			}}
		}
	}

	if opts.DisableBuildCache {
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

// returns the mode used when pushing cached layers to the registry.
//
// "min" only pushes the layers for the final image (no intermediate layers for multi-stage builds)
// "max" pushes all layers into the cache
func getExportMode() (string, error) {
	mode := os.Getenv("EMBEDDED_BUILDER_CACHE_MODE")

	switch {
	case mode == "":
		return defaultCacheMode, nil
	case mode != "min" && mode != "max":
		return "", fmt.Errorf("invalid embedded builder cache mode: %s", mode)
	default:
		return mode, nil
	}
}

func displayProgress(ch chan *controlapi.StatusResponse, logWriter io.Writer) error {
	progressCh := make(chan *bkclient.SolveStatus)

	go func() {
		defer close(progressCh)

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

			progressCh <- &s
		}
	}()

	return progressui.DisplaySolveStatus(context.TODO(), "", nil, logWriter, progressCh)
}

func generateRegistryFunc(registries []config.Registry) (bkimage.CredentialsFn, bkimage.TLSEnabledFn) {
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

	return hostCredentials, matchNonSSL
}
