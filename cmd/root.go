package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/dominodatalab/forge/controllers"
	"github.com/dominodatalab/forge/internal/message"
)

var (
	metricsAddr          string
	enableLeaderElection bool

	messageBroker string
	amqpURI       string
	amqpQueue     string

	brokerOpts *message.Options

	rootCmd = &cobra.Command{
		Use:     "forge",
		Short:   "Kubernetes-native OCI image builder.",
		PreRunE: processBrokerOpts,
		Run: func(cmd *cobra.Command, args []string) {
			controllers.StartManager(metricsAddr, enableLeaderElection, brokerOpts)
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

	rootCmd.Flags().StringVar(&metricsAddr, "metrics-addr", ":8080", "Metrics endpoint will bind to this address")
	rootCmd.Flags().BoolVar(&enableLeaderElection, "enable-leader-election", false, "Enable leader election for controller manager. Enabling this will ensure there is only one active controller manager.")
	rootCmd.Flags().StringVar(&messageBroker, "message-broker", "", fmt.Sprintf("Publish resource state changes to a message broker (supported values: %v)", message.SupportedBrokers))
	rootCmd.Flags().StringVar(&amqpURI, "amqp-uri", "", "AMQP broker connection URI")
	rootCmd.Flags().StringVar(&amqpQueue, "amqp-queue", "", "AMQP broker queue name")
}
