package archive

import (
	"context"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"syscall"
	"testing"

	"github.com/stretchr/testify/assert"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

type errClient struct {
	err error
}

func (e *errClient) Get(_ string) (*http.Response, error) {
	return nil, &url.Error{Err: e.err}
}

type tempError struct {
}

func (tempError) Temporary() bool {
	return true
}

func (tempError) Error() string {
	return "temp"
}

var logger = log.NullLogger{}

func TestFetchAndExtract(t *testing.T) {
	srv := httptest.NewServer(nil)
	defer srv.Close()

	cases := []struct {
		name    string
		archive string
		files   []string
	}{
		{"tarball", "testdata/simple-app.tar", []string{"Dockerfile", "app.py"}},
		{"gzipped-tarball", "testdata/simple-app.tgz", []string{"Dockerfile", "app.py"}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			srv.Config.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				bs, err := ioutil.ReadFile(tc.archive)
				if err != nil {
					t.Fatal(err)
				}
				if _, err := w.Write(bs); err != nil {
					t.Fatal(err)
				}
			})

			ext, err := FetchAndExtract(logger, context.TODO(), srv.URL, 0)
			if err != nil {
				t.Error(err)
			}

			fi, err := os.Stat(ext.Archive)
			if os.IsNotExist(err) {
				t.Errorf("archive was not created %q", ext.Archive)
			}
			if !fi.Mode().IsRegular() {
				t.Errorf("archive is not a regular file %q", ext.Archive)
			}

			var actual []string
			err = filepath.Walk(ext.ContentsDir, func(path string, info os.FileInfo, err error) error {
				if err != nil {
					return err
				}

				if info.IsDir() {
					return nil
				}

				path, _ = filepath.Rel(ext.ContentsDir, path)
				actual = append(actual, path)

				return nil
			})
			if err != nil {
				t.Errorf("failed to list extracted archive files: %v", err)
			}

			assert.ElementsMatch(t, tc.files, actual, "expected archive contents to match")

			os.RemoveAll(ext.RootDir)
		})
	}

	t.Run("unsupported-format", func(t *testing.T) {
		t.SkipNow()
	})
}

func Test_downloadFile(t *testing.T) {
	srv := httptest.NewServer(nil)
	defer srv.Close()

	t.Run("timeout", func(t *testing.T) {
		done, err := downloadFile(logger, &errClient{context.DeadlineExceeded}, "http://my-fake-url", "")
		if done || err != nil {
			t.Errorf("Expected download timeout to retry: %v", err)
		}
	})

	t.Run("temporary failure", func(t *testing.T) {
		done, err := downloadFile(logger, &errClient{tempError{}}, "http://my-fake-url", "")
		if done || err != nil {
			t.Errorf("Expected temporary failure to retry: %v", err)
		}
	})

	t.Run("connection refused", func(t *testing.T) {
		done, err := downloadFile(logger, &errClient{&net.OpError{
			Op:   "dial",
			Net:  "tcp",
			Addr: nil,
			Err: &os.SyscallError{
				Syscall: "connect",
				Err:     syscall.ECONNREFUSED,
			},
		}}, "http://my-fake-url", "")
		if done || err != nil {
			t.Errorf("Expected temporary failure to retry: %v", err)
		}
	})

	cases := []struct {
		statusCode int
		retry      bool
		error      bool
	}{
		{http.StatusGatewayTimeout, true, false},
		{http.StatusBadGateway, true, false},
		{http.StatusServiceUnavailable, true, false},
		{http.StatusBadRequest, true, true},
		{http.StatusInternalServerError, true, true},
		{http.StatusOK, false, false},
	}

	for _, tc := range cases {
		t.Run(fmt.Sprintf("status code: %d", tc.statusCode), func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tc.statusCode)
			}))
			defer srv.Close()

			done, err := downloadFile(logger, srv.Client(), srv.URL, filepath.Join(os.TempDir(), fmt.Sprintf("test-%d.tar", tc.statusCode)))
			if done != tc.retry && (err != nil) != tc.error {
				t.Errorf("Expected status code %d (retry=%v, error=%v): got (done=%v, error=%v)", tc.statusCode, tc.retry, tc.error, done, err)
			}
		})
	}
}
