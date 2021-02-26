package util

import (
	"fmt"
	"io/ioutil"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

// Succeeds when target is an existing directory.
func TestAssertDir_isDir(t *testing.T) {
	tempDir, err := ioutil.TempDir("", "")
	if err != nil {
		t.Errorf("error creating temporary directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	testDir := tempDir + "/test-dir"

	if err := os.Mkdir(testDir, 0755); err != nil {
		t.Errorf("error creating diretory: %v", err)
	}

	err = AssertDir(testDir)

	assert.Equal(t, nil, err)
}

// Fails with error message when target is an existing file.
func TestAssertDir_isFile(t *testing.T) {
	tempDir, err := ioutil.TempDir("", "")
	if err != nil {
		t.Errorf("error creating temporary directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	testFile := tempDir + "/test-file"

	if file, err := os.Create(testFile); err != nil {
		t.Errorf("error creating file: %v", err)
	} else {
		defer file.Close()
	}

	err = AssertDir(testFile)

	expected := fmt.Errorf("%q is not a directory", testFile)
	assert.Equal(t, expected, err)
}

// Fails with PathError when target does not exist.
func TestAssertDir_notFound(t *testing.T) {
	tempDir, err := ioutil.TempDir("", "")
	if err != nil {
		t.Errorf("error creating temporary directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	testFile := tempDir + "/test-file"

	err = AssertDir(testFile)

	assert.IsType(t, &os.PathError{}, err)
}
