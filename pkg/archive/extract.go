package archive

import (
	"archive/tar"
	"bufio"
	"compress/gzip"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"

	"github.com/h2non/filetype"
)

type mimeType string

const (
	mimeTypeTar  = mimeType("application/x-tar")
	mimeTypeGzip = mimeType("application/gzip")
)

type Extractor func(url string) (*Extraction, error)

type Extraction struct {
	RootDir     string
	Archive     string
	ContentsDir string
}

func FetchAndExtract(url string) (*Extraction, error) {
	wd, err := ioutil.TempDir("", "forge-")
	if err != nil {
		return nil, err
	}

	archive := filepath.Join(wd, "archive")
	if err := downloadFile(url, archive); err != nil {
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

func downloadFile(url, fp string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("file download failed with status %d", resp.StatusCode)
	}

	out, err := os.Create(fp)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	return err
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
