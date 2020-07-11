package preparer

import (
	"os"
	"path/filepath"

	"github.com/go-logr/logr"
	"github.com/hashicorp/go-hclog"
)

type pluginLoader = func(string, hclog.Logger) (*Plugin, error)

var DefaultPluginLoader = NewPreparerPlugin

func LoadPlugins(preparerPluginsPath string, logger logr.Logger) (preparerPlugins []*Plugin, err error) {
	return loadPlugins(preparerPluginsPath, &logrLogger{logger}, DefaultPluginLoader)
}

func loadPlugins(preparerPluginsPath string, logger hclog.Logger, loader pluginLoader) (preparerPlugins []*Plugin, err error) {
	if preparerPluginsPath == "" {
		return
	}

	// If the default path does not exist, just return and continue
	if _, pathErr := os.Stat(preparerPluginsPath); os.IsNotExist(pathErr) {
		return
	}

	err = filepath.Walk(preparerPluginsPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() {
			preparerPlugin, err := loader(path, logger)
			if err != nil {
				return err
			}
			preparerPlugins = append(preparerPlugins, preparerPlugin)
		}

		return nil
	})

	return
}
