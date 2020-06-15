package preparer

type Preparer interface {
	Prepare(string, map[string]string) error
	Cleanup() error
}

type Arguments struct {
	ContextPath string
	PluginData  map[string]string
}
