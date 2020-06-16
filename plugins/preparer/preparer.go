package preparer

import (
	"os"
	"os/exec"

	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/go-plugin"
	"github.com/pkg/errors"
)

var handshakeConfig = plugin.HandshakeConfig{
	ProtocolVersion:  1,
	MagicCookieKey:   os.Getenv("FORGE_PREPARER_PLUGIN_MAGIC_KEY"),
	MagicCookieValue: os.Getenv("FORGE_PREPARER_PLUGIN_MAGIC_VALUE"),
}

func NewPreparerPlugin(location string) (*Plugin, error) {
	logger := hclog.New(&hclog.LoggerOptions{
		Name:   "plugins",
		Output: os.Stdout,
		// TODO configurable level, from LOG_LEVEL?
		Level: hclog.Debug,
	})

	client := plugin.NewClient(&plugin.ClientConfig{
		HandshakeConfig: handshakeConfig,
		Plugins: map[string]plugin.Plugin{
			"preparer": &Plugin{},
		},
		Cmd:    exec.Command(location),
		Logger: logger,
	})

	rpcClient, err := client.Client()
	if err != nil {
		return nil, errors.Wrapf(err, "rpc client create failed for %q", location)
	}

	raw, err := rpcClient.Dispense("preparer")
	if err != nil {
		return nil, errors.Wrapf(err, "cannot dispense rpc client for %q", location)
	}

	return &Plugin{
		client:   client,
		Preparer: raw.(Preparer),
	}, nil
}
