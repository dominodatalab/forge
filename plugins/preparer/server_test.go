package preparer

import (
	"reflect"
	"testing"

	"github.com/pkg/errors"
)

type testPreparer struct {
	err               error
	calledContextPath string
	calledPluginData  map[string]string
	calledCleanup     bool
}

func (t *testPreparer) Prepare(contextPath string, pluginData map[string]string) error {
	t.calledContextPath = contextPath
	t.calledPluginData = pluginData
	return t.err
}

func (t *testPreparer) Cleanup() error {
	t.calledCleanup = true
	return t.err
}

func Test_preparerServer_Prepare(t *testing.T) {
	testPreparer := &testPreparer{}
	server := &preparerServer{testPreparer}

	contextPath := "/test/path"
	pluginData := map[string]string{"test": "data"}
	arguments := &Arguments{contextPath, pluginData}

	var errStr string
	err := server.Prepare(arguments, &errStr)

	if err != nil {
		t.Errorf("preparer returned an error unexpectedly: %v", err)
		return
	}

	if testPreparer.calledContextPath != contextPath || !reflect.DeepEqual(testPreparer.calledPluginData, pluginData) {
		t.Errorf("preparer arguments %v were not called on preparer: %v", arguments, testPreparer)
		return
	}

	if errStr != "" {
		t.Errorf("preparer error unexpectedly returned, got %q", errStr)
		return
	}

	expectedErrStr := "this is a test error"
	testPreparer.err = errors.New(expectedErrStr)
	err = server.Prepare(arguments, &errStr)
	if err != nil {
		t.Errorf("preparer returned an error unexpectedly: %v", err)
		return
	}

	if errStr != "this is a test error" {
		t.Errorf("preparer error was not successfully returned, got %q; wanted %q", errStr, expectedErrStr)
	}
}

func Test_preparerServer_Cleanup(t *testing.T) {
	testPreparer := &testPreparer{}
	server := &preparerServer{testPreparer}

	var errStr string
	err := server.Cleanup(&Arguments{}, &errStr)

	if err != nil {
		t.Errorf("preparer returned an error unexpectedly: %v", err)
		return
	}

	if !testPreparer.calledCleanup {
		t.Error("preparer Cleanup() did not get called")
		return
	}

	if errStr != "" {
		t.Errorf("preparer error unexpectedly returned, got %q", errStr)
		return
	}

	expectedErrStr := "this is a test error"
	testPreparer.err = errors.New(expectedErrStr)
	err = server.Cleanup(&Arguments{}, &errStr)
	if err != nil {
		t.Errorf("preparer returned an error unexpectedly: %v", err)
		return
	}

	if errStr != "this is a test error" {
		t.Errorf("preparer error was not successfully returned, got %q; wanted %q", errStr, expectedErrStr)
	}
}
