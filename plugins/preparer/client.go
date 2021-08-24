package preparer

import (
	"github.com/pkg/errors"
)

var _ Preparer = &preparerClient{}

const (
	prepareServiceMethod = "Plugin.Prepare"
	cleanupServiceMethod = "Plugin.Cleanup"
)

type rpcClient interface {
	Call(serviceMethod string, args interface{}, reply interface{}) error
}

type preparerClient struct {
	client rpcClient
}

func (p *preparerClient) Prepare(contextPath string, pluginData map[string]string) error {
	var errStr string

	err := p.client.Call(prepareServiceMethod, &Arguments{contextPath, pluginData}, &errStr)
	if err == nil && errStr != "" {
		err = errors.New(errStr)
	}

	return errors.Wrapf(err, "failed to prepare %s", contextPath)
}

func (p *preparerClient) Cleanup() error {
	var errStr string

	err := p.client.Call(cleanupServiceMethod, &Arguments{}, &errStr)
	if err == nil && errStr != "" {
		err = errors.New(errStr)
	}

	return errors.Wrap(err, "failed to cleanup")
}
