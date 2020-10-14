package buildjob

type Config struct {
	ResourceName        string
	ResourceNamespace   string
	PreparerPluginsPath string
	EnableLayerCaching  bool
	Debug               bool
}
