package preparer

import (
	"errors"
	"net/rpc"
)

type rpcClient struct {
	client *rpc.Client
}

var _ Preparer = &rpcClient{}

func (p *rpcClient) Prepare(contextPath string, pluginData map[string]string) error {
	var errStr string
	if rpcError := p.client.Call("Plugin.Prepare", &Arguments{contextPath, pluginData}, &errStr); rpcError != nil {
		return rpcError
	}

	if errStr != "" {
		return errors.New(errStr)
	}

	return nil
}

func (p *rpcClient) Cleanup() error {
	var errStr string
	if rpcError := p.client.Call("Plugin.Cleanup", &Arguments{}, &errStr); rpcError != nil {
		return rpcError
	}

	if errStr != "" {
		return errors.New(errStr)
	}

	return nil
}
