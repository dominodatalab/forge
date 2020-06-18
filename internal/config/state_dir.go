package config

import (
	"os"
	"path/filepath"
	"strings"
)

func GetStateDir() string {
	//  pam_systemd sets XDG_RUNTIME_DIR but not other dirs.
	if xdgDataHome := os.Getenv("XDG_DATA_HOME"); xdgDataHome != "" {
		dir := strings.Split(xdgDataHome, ":")[0]
		return filepath.Join(dir)
	}

	if home := os.Getenv("HOME"); home != "" {
		return filepath.Join(home, ".local", "share", "forge")
	}

	return "/tmp/forge"
}
