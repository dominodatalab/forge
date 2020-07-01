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
possible, this controller can be configured to push state updates to any AMQP message broker.`

	examples = `
# Watch for ContainerImageBuild resources in your namespace
forge --namespace <my-ns>

# Publish status updates to an AMQP message broker
forge --message-broker amqp --amqp-uri amqp://<user>:<pass>@<host>:<port/<path> --amqp-queue <queue-name>

# Leverage one or more plugins for pre-processing a context prior to build
forge --preparer-plugins-path /plugins/installed/here`
)

var (
	debug bool

	namespace            string
	metricsAddr          string
	enableLeaderElection bool
	messageBroker        string
	amqpURI              string
	amqpQueue            string
	preparerPluginsPath  string
	brokerOpts           *message.Options

	rootCmd = &cobra.Command{
		Use:     "forge",
		Long:    description,
		Example: examples,
		PreRunE: processBrokerOpts,
		Run: func(cmd *cobra.Command, args []string) {
			controllers.StartManager(namespace, metricsAddr, enableLeaderElection, brokerOpts, preparerPluginsPath, debug)
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

	rootCmd.Flags().StringVar(&namespace, "namespace", "default", "Watch for objects in desired namespace")
	rootCmd.Flags().StringVar(&metricsAddr, "metrics-addr", ":8080", "Metrics endpoint will bind to this address")
	rootCmd.Flags().BoolVar(&enableLeaderElection, "enable-leader-election", false, "Enable leader election for controller manager. Enabling this will ensure there is only one active controller manager.")
	rootCmd.Flags().StringVar(&messageBroker, "message-broker", "", fmt.Sprintf("Publish resource state changes to a message broker (supported values: %v)", message.SupportedBrokers))
	rootCmd.Flags().StringVar(&amqpURI, "amqp-uri", "", "AMQP broker connection URI")
	rootCmd.Flags().StringVar(&amqpQueue, "amqp-queue", "", "AMQP broker queue name")
	rootCmd.Flags().StringVar(&preparerPluginsPath, "preparer-plugins-path", path.Join(config.GetStateDir(), "plugins"), "Path to specific preparer plugins or directory to load them from")
	rootCmd.Flags().BoolVar(&debug, "debug", false, "Enabled verbose logging")
}
