package example

import (
	"os"

	"github.com/hashicorp/go-plugin"

	forge "github.com/dominodatalab/forge/plugins/preparer"
)

var handshakeConfig = plugin.HandshakeConfig{
	ProtocolVersion:  1,
	MagicCookieKey:   os.Getenv("FORGE_PREPARER_PLUGIN_MAGIC_KEY"),
	MagicCookieValue: os.Getenv("FORGE_PREPARER_PLUGIN_MAGIC_VALUE"),
}

type MyPlugin struct{}

func (*MyPlugin) Prepare(contextPath string, pluginData map[string]string) error {
	// Prepare is called with a context path and any custom plugin data.
	return nil
}

func (*MyPlugin) Cleanup() error {
	// Any cleanup is run after the build is finished (successful or errored).
	return nil
}

func main() {
	plugin.Serve(&plugin.ServeConfig{
		HandshakeConfig: handshakeConfig,
		Plugins: map[string]plugin.Plugin{
			"preparer": &forge.Plugin{Preparer: &MyPlugin{}},
		},
	})
}
