package preparer

import (
	"os"
	"path/filepath"
)

func LoadPlugins(preparerPluginsPath string) ([]*Plugin, error) {
	var preparerPlugins []*Plugin

	err := filepath.Walk(preparerPluginsPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() {
			preparerPlugin, err := NewPreparerPlugin(path)
			if err != nil {
				return err
			}
			preparerPlugins = append(preparerPlugins, preparerPlugin)
		}

		return nil
	})

	return preparerPlugins, err
}
