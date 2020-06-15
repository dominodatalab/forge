package preparer

type rpcServer struct {
	Preparer
}

func (p *rpcServer) Prepare(args *Arguments, errStr *string) error {
	if err := p.Preparer.Prepare(args.ContextPath, args.PluginData); err != nil {
		*errStr = err.Error()
	}
	return nil
}

func (p *rpcServer) Cleanup(_ *Arguments, errStr *string) error {
	if err := p.Preparer.Cleanup(); err != nil {
		*errStr = err.Error()
	}
	return nil
}
