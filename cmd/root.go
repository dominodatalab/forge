package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/dominodatalab/forge/controllers"
)

var (
	metricsAddr          string
	enableLeaderElection bool

	rootCmd = &cobra.Command{
		Use:   "forge",
		Short: "Kubernetes-native OCI image builder.",
		Run: func(cmd *cobra.Command, args []string) {
			controllers.StartManager(metricsAddr, enableLeaderElection)
		},
	}
)

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.Flags().SortFlags = false
	rootCmd.Flags().StringVar(&metricsAddr, "metrics-addr", ":8080", "Metrics endpoint will bind to this address")
	rootCmd.Flags().BoolVar(&enableLeaderElection, "enable-leader-election", false, "Enable leader election for controller manager. Enabling this will ensure there is only one active controller manager.")
}
