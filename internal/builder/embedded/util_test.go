package embedded

import (
	"os"
	"testing"
)

func TestDriver_getExportMode(t *testing.T) {
	t.Run("default", func(t *testing.T) {
		mode, err := getExportMode()
		if err != nil {
			t.Fatal(err)
		}
		if mode != "max" {
			t.Errorf("expected %s, got %s", "max", mode)
		}
	})

	t.Run("env_override", func(t *testing.T) {
		validModes := []string{"min", "max"}
		for _, expected := range validModes {
			os.Setenv("EMBEDDED_BUILDER_CACHE_MODE", expected)

			actual, err := getExportMode()
			if err != nil {
				t.Fatal(err)
			}
			if actual != expected {
				t.Errorf("expected %s, got %s", expected, actual)
			}

			os.Unsetenv("EMBEDDED_BUILDER_CACHE_MODE")
		}
	})

	t.Run("invalid", func(t *testing.T) {
		os.Setenv("EMBEDDED_BUILDER_CACHE_MODE", "steve-o")
		defer func() {
			os.Unsetenv("EMBEDDED_BUILDER_CACHE_MODE")
		}()

		if _, err := getExportMode(); err == nil {
			t.Error("expected err, got none")
		}
	})
}
