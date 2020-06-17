package preparer

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestLoadPlugins(t *testing.T) {
	var emptyPlugins []*Plugin

	emptyPluginDir, err := ioutil.TempDir("", "")
	if err != nil {
		t.Error(err)
		return
	}
	defer os.RemoveAll(emptyPluginDir)

	pluginDir, err := ioutil.TempDir("", "")
	if err != nil {
		t.Error(err)
		return
	}
	defer os.RemoveAll(pluginDir)

	pluginFile := filepath.Join(pluginDir, "my-plugin")
	if err := ioutil.WriteFile(pluginFile, []byte("this-is-a-plugin"), 0664); err != nil {
		t.Error(err)
		return
	}

	tests := []struct {
		name            string
		path            string
		expectedPlugins []*Plugin
		wantErr         bool
	}{
		{
			"does_not_exist",
			"/this/does/not/exist/hopefully",
			emptyPlugins,
			false,
		},
		{
			"exists_but_empty",
			emptyPluginDir,
			emptyPlugins,
			false,
		},
		{
			"exists_with_plugin",
			pluginDir,
			[]*Plugin{{}},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			DefaultPluginLoader = func(path string) (*Plugin, error) {
				return &Plugin{}, nil
			}
			defer func() {
				DefaultPluginLoader = NewPreparerPlugin
			}()

			gotPreparerPlugins, err := LoadPlugins(tt.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("LoadPlugins() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !reflect.DeepEqual(gotPreparerPlugins, tt.expectedPlugins) {
				t.Errorf("LoadPlugins() gotPreparerPlugins = %v, want %v", gotPreparerPlugins, tt.expectedPlugins)
			}
		})
	}
}
