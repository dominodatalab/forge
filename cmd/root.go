package cmd

import (
	"fmt"
	"os"
	"path"
	"strings"

	"github.com/spf13/cobra"

	"github.com/dominodatalab/forge/controllers"
	"github.com/dominodatalab/forge/internal/config"
	"github.com/dominodatalab/forge/internal/message"
)

const (
	description = `Kubernetes-native OCI image builder.

Forge is a Kubernetes controller that builds and pushes OCI-compliant images to one or more distribution registries.
Communication with the controller is achieved via the ContainerImageBuild CRD defined by the project. Forge will watch
for these resources, launch an image build using the directives provided therein, and update the resource status with
relevant information such as build state, errors and the final location(s) of the 

If you need to run preparation steps against a context directory prior to a build, then you can configure one or more
plugins. This allows users to hook into the build process and add/modify/delete files according to their business
workflows.

Ideally, state change consumers should set a watch on their ContainerImageBuild resources for updates. When this is not
possible, this controller can be configured to push state updates to any AMQP message broker.

Image layers can be exported and stored inside the target registry after a build. This shared cache will then be used by
all controller workers. By default, the embedded image builder uses a "max" mode to ensure all intermediate and final
image layers are exported. You can override this behavior using the EMBEDDED_BUILDER_CACHE_MODE environment variable.
Acceptable values include "min" and "max".`

	examples = `
# Watch for ContainerImageBuild resources in your namespace
forge --namespace <my-ns>

# Publish status updates to an AMQP message broker
forge --message-broker amqp --amqp-uri amqp://<user>:<pass>@<host>:<port/<path> --amqp-queue <queue-name>

# Leverage one or more plugins for pre-processing a context prior to build
forge --preparer-plugins-path /plugins/installed/here

# Enable image build layer caching
forge --enable-layer-caching`
)

var (
	debug bool

	buildJobImage        string
	namespace            string
	metricsAddr          string
	enableLeaderElection bool
	messageBroker        string
	amqpURI              string
	amqpQueue            string
	preparerPluginsPath  string
	enableLayerCaching   bool
	brokerOpts           *message.Options
	grantFullPrivilege   bool
	customCASecret       string

	rootCmd = &cobra.Command{
		Use:               "forge",
		Long:              description,
		Example:           examples,
		PersistentPreRunE: processBrokerOpts,
		Run: func(cmd *cobra.Command, args []string) {
			cfg := controllers.Config{
				BuildJobImage:         buildJobImage,
				BuildJobFullPrivilege: grantFullPrivilege,
				CustomCASecret:        customCASecret,
				Namespace:             namespace,
				MetricsAddr:           metricsAddr,
				EnableLeaderElection:  enableLeaderElection,
				BrokerOpts:            brokerOpts,
				PreparerPluginsPath:   preparerPluginsPath,
				EnableLayerCaching:    enableLayerCaching,
				Debug:                 debug,
			}
			controllers.StartManager(cfg)
		},
	}
)

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func processBrokerOpts(cmd *cobra.Command, args []string) error {
	if messageBroker == "" {
		return nil
	}

	brokerOpts = &message.Options{
		Broker:    message.Broker(strings.ToLower(messageBroker)),
		AmqpURI:   amqpURI,
		AmqpQueue: amqpQueue,
	}
	return message.ValidateOpts(brokerOpts)
}

func init() {
	rootCmd.Flags().SortFlags = false

	// main command flags
	rootCmd.Flags().StringVar(&namespace, "namespace", "default", "Watch for objects in desired namespace")
	rootCmd.Flags().StringVar(&metricsAddr, "metrics-addr", ":8080", "Metrics endpoint will bind to this address")
	rootCmd.Flags().StringVar(&buildJobImage, "builder-job-image", buildJobImage, "Image used to launch build jobs. This typically should be the same as the controller.")
	rootCmd.Flags().BoolVar(&enableLeaderElection, "enable-leader-election", false, "Enable leader election for controller manager. Enabling this will ensure there is only one active controller manager.")
	rootCmd.Flags().BoolVar(&grantFullPrivilege, "full-privilege", false, "Run builds jobs using a privileged root user")
	rootCmd.Flags().StringVar(&customCASecret, "custom-ca-secret", "", "Secret container custom CA certificates for distribution registries")

	// leveraged by both main and build commands
	rootCmd.PersistentFlags().StringVar(&messageBroker, "message-broker", "", fmt.Sprintf("Publish resource state changes to a message broker (supported values: %v)", message.SupportedBrokers))
	rootCmd.PersistentFlags().StringVar(&amqpURI, "amqp-uri", "", "AMQP broker connection URI")
	rootCmd.PersistentFlags().StringVar(&amqpQueue, "amqp-queue", "", "AMQP broker queue name")
	rootCmd.PersistentFlags().StringVar(&preparerPluginsPath, "preparer-plugins-path", path.Join(config.GetStateDir(), "plugins"), "Path to specific preparer plugins or directory to load them from")
	rootCmd.PersistentFlags().BoolVar(&enableLayerCaching, "enable-layer-caching", false, "Enable image layer caching")
	rootCmd.PersistentFlags().BoolVar(&debug, "debug", false, "Enabled verbose logging")
}
