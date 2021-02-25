package util

import (
	"fmt"
	"os"
)

func AssertDir(path string) error {
	fi, err := os.Stat(path)
	if err != nil {
		return err
	}
	if !fi.IsDir() {
		return fmt.Errorf("%q is not a directory", path)
	}

	return nil
}
