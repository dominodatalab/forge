package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"strings"

	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"

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

	defaultBuildJobCAImage = "quay.io/domino/forge-init-ca:v1.0.0"
)

var (
	debug bool

	buildJobImage               string
	buildJobCAImage             string
	buildJobLabels              map[string]string
	buildJobAnnotations         map[string]string
	buildJobCustomCASecret      string
	buildJobGrantFullPrivilege  bool
	buildAdvancedConfigFilename string

	namespace            string
	metricsAddr          string
	enableLeaderElection bool
	messageBroker        string
	amqpURI              string
	amqpQueue            string
	preparerPluginsPath  string
	enableLayerCaching   bool
	brokerOpts           *message.Options

	advCfg = &advancedConfig{}

	rootCmd = &cobra.Command{
		Use:               "forge",
		Long:              description,
		Example:           examples,
		PreRunE:           processAdvancedConfig,
		PersistentPreRunE: processBrokerOpts,
		Run: func(cmd *cobra.Command, args []string) {
			cfg := controllers.ControllerConfig{
				Debug:                debug,
				Namespace:            namespace,
				MetricsAddr:          metricsAddr,
				EnableLeaderElection: enableLeaderElection,

				JobConfig: &controllers.BuildJobConfig{
					Image:              buildJobImage,
					CAImage:            buildJobCAImage,
					CustomCASecret:     buildJobCustomCASecret,
					PreparerPluginPath: preparerPluginsPath,
					Labels:             buildJobLabels,
					Annotations:        buildJobAnnotations,
					GrantFullPrivilege: buildJobGrantFullPrivilege,
					EnableLayerCaching: enableLayerCaching,
					BrokerOpts:         brokerOpts,
					EnvVar:             advCfg.Env,
					Volumes:            advCfg.Volumes,
					VolumeMounts:       advCfg.VolumeMounts,
				},
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

type advancedConfig struct {
	Env          []corev1.EnvVar
	Volumes      []corev1.Volume      `json:"volumes"`
	VolumeMounts []corev1.VolumeMount `json:"volumeMounts"`
}

func processAdvancedConfig(cmd *cobra.Command, args []string) error {
	if buildAdvancedConfigFilename == "" {
		return nil
	}

	bs, err := ioutil.ReadFile(buildAdvancedConfigFilename)
	if err != nil {
		return err
	}

	dec := json.NewDecoder(bytes.NewBuffer(bs))
	dec.DisallowUnknownFields()
	return dec.Decode(advCfg)
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
	rootCmd.Flags().BoolVar(&enableLeaderElection, "enable-leader-election", false, "Enable leader election for controller manager. Enabling this will ensure there is only one active controller manager.")

	rootCmd.Flags().StringVar(&buildJobImage, "build-job-image", buildJobImage, "Image used to launch build jobs. This typically should be the same as the controller.")
	rootCmd.Flags().StringVar(&buildJobCAImage, "build-job-ca-image", defaultBuildJobCAImage, "Image used to initialize SSL certificates using a custom CA. You should not have to override this.")
	rootCmd.Flags().StringToStringVar(&buildJobLabels, "build-job-labels", nil, "Additional labels added to build job pods")
	rootCmd.Flags().StringToStringVar(&buildJobAnnotations, "build-job-annotations", nil, "Additional annotations added to build job pods")
	rootCmd.Flags().StringVar(&buildJobCustomCASecret, "build-job-custom-ca", "", "Secret container custom CA certificates for distribution registries")
	rootCmd.Flags().BoolVar(&buildJobGrantFullPrivilege, "build-job-full-privilege", false, "Run builds jobs using a privileged root user")
	rootCmd.Flags().StringVar(&buildAdvancedConfigFilename, "build-job-advanced-config", "", "Add volumes, volume mounts and environment variables to your build jobs using a JSON file")

	// leveraged by both main and build commands
	rootCmd.PersistentFlags().StringVar(&messageBroker, "message-broker", "", fmt.Sprintf("Publish resource state changes to a message broker (supported values: %v)", message.SupportedBrokers))
	rootCmd.PersistentFlags().StringVar(&amqpURI, "amqp-uri", "", "AMQP broker connection URI")
	rootCmd.PersistentFlags().StringVar(&amqpQueue, "amqp-queue", "", "AMQP broker queue name")
	rootCmd.PersistentFlags().StringVar(&preparerPluginsPath, "preparer-plugins-path", path.Join(config.GetStateDir(), "plugins"), "Path to specific preparer plugins or directory to load them from")
	rootCmd.PersistentFlags().BoolVar(&enableLayerCaching, "enable-layer-caching", false, "Enable image layer caching")
	rootCmd.PersistentFlags().BoolVar(&debug, "debug", false, "Enabled verbose logging")
}
