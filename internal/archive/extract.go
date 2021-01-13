package archive

import (
	"archive/tar"
	"bufio"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"time"

	"github.com/go-logr/logr"
	"github.com/h2non/filetype"
	"k8s.io/apimachinery/pkg/util/wait"
)

type mimeType string

const (
	mimeTypeTar  = mimeType("application/x-tar")
	mimeTypeGzip = mimeType("application/gzip")
)

var defaultBackoff = wait.Backoff{
	Duration: time.Second,
	Factor:   2,
	Steps:    10,
	Jitter:   0.1,
	Cap:      30 * time.Second,
}

type fileDownloader interface {
	Get(string) (*http.Response, error)
}

type Extractor func(logr.Logger, context.Context, string, time.Duration) (*Extraction, error)

type Extraction struct {
	RootDir     string
	Archive     string
	ContentsDir string
}

func FetchAndExtract(log logr.Logger, ctx context.Context, url string, timeout time.Duration) (*Extraction, error) {
	if timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}

	wd, err := ioutil.TempDir("", "forge-")
	if err != nil {
		return nil, err
	}

	archive := filepath.Join(wd, "archive")

	err = wait.ExponentialBackoff(defaultBackoff, func() (bool, error) {
		// TODO in client-go v0.21.0 ExponentialBackoffWithContext can handle this for us
		select {
		case <-ctx.Done():
			return false, ctx.Err()
		default:
		}

		return downloadFile(log, http.DefaultClient, url, archive)
	})
	if err != nil {
		return nil, err
	}

	ct, err := getFileContentType(archive)
	if err != nil {
		return nil, err
	}
	if ct != mimeTypeGzip && ct != mimeTypeTar {
		return nil, fmt.Errorf("unsupported file type %q", ct)
	}

	dest := filepath.Join(wd, "extracted")
	if err := os.Mkdir(dest, 0755); err != nil {
		return nil, err
	}
	if err := extract(archive, ct, dest); err != nil {
		return nil, err
	}

	return &Extraction{
		RootDir:     wd,
		Archive:     archive,
		ContentsDir: dest,
	}, nil
}

func retryable(err *url.Error) bool {
	if err.Timeout() || err.Temporary() {
		return true
	}

	// If we get any sort of operational error before an HTTP response we
	// retry it. Generally have seen this with ECONNREFUSED.
	if _, ok := err.Err.(*net.OpError); ok {
		return true
	}

	return false
}

// downloadFile takes a file URL and local location to download it to.
// It returns "done" (retryable or not) and an error.
func downloadFile(log logr.Logger, c fileDownloader, fileUrl, fp string) (bool, error) {
	resp, err := c.Get(fileUrl)
	if err != nil {
		if urlError, ok := err.(*url.Error); ok && retryable(urlError) {
			log.Error(urlError, "Received temporary or transient error while fetching context, will attempt to retry", "url", fileUrl, "file", fp)
			return false, nil
		}

		return false, err
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusBadGateway, http.StatusGatewayTimeout, http.StatusServiceUnavailable:
		log.Info("Received transient status code while fetching context, will attempt to retry", "url", fileUrl, "file", fp, "code", resp.StatusCode)
		return false, nil
	case http.StatusOK:
	default:
		return false, fmt.Errorf("file download failed with status %d", resp.StatusCode)
	}

	out, err := os.Create(fp)
	if err != nil {
		return false, err
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	return true, err
}

func getFileContentType(fp string) (ct mimeType, err error) {
	f, err := os.Open(fp)
	if err != nil {
		return
	}
	defer f.Close()

	buf := make([]byte, 512)
	if _, err = f.Read(buf); err != nil {
		return
	}

	kind, err := filetype.Match(buf)
	if err != nil {
		return
	}

	return mimeType(kind.MIME.Value), nil
}

func extract(fp string, ct mimeType, dst string) error {
	f, err := os.Open(fp)
	if err != nil {
		return err
	}
	defer f.Close()

	var r io.Reader
	if ct == mimeTypeGzip {
		gzr, err := gzip.NewReader(f)
		if err != nil {
			return err
		}
		defer gzr.Close()

		r = gzr
	} else {
		r = bufio.NewReader(f)
	}

	tr := tar.NewReader(r)

	for {
		header, err := tr.Next()

		switch {
		case err == io.EOF:
			return nil
		case err != nil:
			return err
		case header == nil:
			continue
		}

		target := filepath.Join(dst, header.Name)

		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0755); err != nil {
				return err
			}
		case tar.TypeReg:
			f, err := os.OpenFile(target, os.O_CREATE|os.O_RDWR, os.FileMode(header.Mode))
			if err != nil {
				return err
			}

			_, err = io.Copy(f, tr)
			f.Close()

			if err != nil {
				return err
			}
		}
	}
}
