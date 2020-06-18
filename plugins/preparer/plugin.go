package preparer

import (
	"net/rpc"

	"github.com/hashicorp/go-plugin"
)

type Plugin struct {
	client *plugin.Client
	Preparer
}

func (p *Plugin) Server(*plugin.MuxBroker) (interface{}, error) {
	return &preparerServer{p.Preparer}, nil
}

func (p *Plugin) Client(b *plugin.MuxBroker, c *rpc.Client) (interface{}, error) {
	return &preparerClient{client: c}, nil
}

func (p *Plugin) Kill() {
	p.client.Kill()
}
