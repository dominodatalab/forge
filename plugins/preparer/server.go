package preparer

type preparerServer struct {
	Preparer
}

func (p *preparerServer) Prepare(args *Arguments, errStr *string) error {
	if err := p.Preparer.Prepare(args.ContextPath, args.PluginData); err != nil {
		*errStr = err.Error()
	}
	return nil
}

func (p *preparerServer) Cleanup(_ *Arguments, errStr *string) error {
	if err := p.Preparer.Cleanup(); err != nil {
		*errStr = err.Error()
	}
	return nil
}
