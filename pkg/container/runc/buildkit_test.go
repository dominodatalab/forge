package runc

//import (
//	"os"
//	"testing"
//
//	"github.com/moby/buildkit/session"
//	"github.com/moby/buildkit/session/auth/authprovider"
//
//	"github.com/dominodatalab/forge/pkg/archive"
//	"github.com/dominodatalab/forge/pkg/container/config"
//	"github.com/moby/buildkit/client"
//	"github.com/stretchr/testify/assert"
//)
//
//var fakeExtractor = func(string) (*archive.Extraction, error) {
//	return &archive.Extraction{
//		RootDir:     "/test",
//		Archive:     "/test/tarball",
//		ContentsDir: "/test/tarball-contents/",
//	}, nil
//}
//
//func TestBuilder_PrepareSolveOpt(t *testing.T) {
//	builder := builder{extractor: fakeExtractor}
//	opts := config.BuildOptions{
//		RegistryURL: "my-registry:5000",
//		ImageName:   "my-image",
//		Context:     "https://my.remote/context",
//	}
//	expected := &client.SolveOpt{
//		Frontend: "dockerfile.v0",
//		FrontendAttrs: map[string]string{
//			"filename": "Dockerfile",
//		},
//		LocalDirs: map[string]string{
//			"context":    "/test/tarball-contents/",
//			"dockerfile": "/test/tarball-contents/",
//		},
//		Session: []session.Attachable{authprovider.NewDockerAuthProvider(os.Stderr)},
//		Exports: []client.ExportEntry{
//			{
//				Type: "image",
//				Attrs: map[string]string{
//					"name":              "my-registry:5000/my-image",
//					"push":              "true",
//					"registry.insecure": "false",
//				},
//			},
//		},
//	}
//
//	t.Run("default", func(t *testing.T) {
//		actual, err := builder.prepareSolveOpt(opts)
//
//		assert.NoError(t, err)
//		assert.Equal(t, expected, actual)
//	})
//
//	t.Run("no_cache", func(t *testing.T) {
//
//	})
//
//	t.Run("insecure_registry", func(t *testing.T) {
//
//	})
//
//	t.Run("labels", func(t *testing.T) {
//
//	})
//
//	t.Run("build_args", func(t *testing.T) {
//
//	})
//
//	t.Run("context_err", func(t *testing.T) {
//
//	})
//}
