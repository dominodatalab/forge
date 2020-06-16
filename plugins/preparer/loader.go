package preparer

import (
	"os"
	"path/filepath"
)

func LoadPlugins(preparerPluginsPath string) (preparerPlugins []*Plugin, err error) {
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
			preparerPlugin, err := NewPreparerPlugin(path)
			if err != nil {
				return err
			}
			preparerPlugins = append(preparerPlugins, preparerPlugin)
		}

		return nil
	})

	return
}
