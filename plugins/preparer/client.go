package preparer

import (
	"net/rpc"

	"github.com/pkg/errors"
)

var _ Preparer = &rpcClient{}

const (
	prepareServiceMethod = "Plugin.Prepare"
	cleanupServiceMethod = "Plugin.Cleanup"
)

type rpcClient struct {
	client *rpc.Client
}

func (p *rpcClient) Prepare(contextPath string, pluginData map[string]string) error {
	var errStr string

	err := p.client.Call(prepareServiceMethod, &Arguments{contextPath, pluginData}, &errStr)
	if err == nil && errStr != "" {
		err = errors.New(errStr)
	}

	return errors.Wrapf(err, "failed to prepare %s with %v", contextPath, pluginData)
}

func (p *rpcClient) Cleanup() error {
	var errStr string

	err := p.client.Call(cleanupServiceMethod, &Arguments{}, &errStr)
	if err == nil && errStr != "" {
		err = errors.New(errStr)
	}

	return errors.Wrap(err, "failed to cleanup")
}
