package cmd

import (
	"io/ioutil"

	"github.com/spf13/cobra"

	"github.com/dominodatalab/forge/internal/buildjob"
)

var (
	resourceName      string
	resourceNamespace string

	buildCmd = &cobra.Command{
		Use:   "build",
		Short: "Launch a single OCI image build",
		Long:  "fill in the details",
		PreRun: func(cmd *cobra.Command, args []string) {
			// attempt to load "current namespace" when running inside k8s
			bs, err := ioutil.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/namespace")
			if err != nil {
				return
			}

			// set that value as the ns where the build job should search for containerimagebuild resources
			if err := cmd.Flags().Set("resource-namespace", string(bs)); err != nil {
				panic(err)
			}
		},
		Run: func(cmd *cobra.Command, args []string) {
			cfg := buildjob.Config{
				ResourceName:        resourceName,
				ResourceNamespace:   resourceNamespace,
				BrokerOpts:          brokerOpts,
				PreparerPluginsPath: preparerPluginsPath,
				EnableLayerCaching:  enableLayerCaching,
				Debug:               debug,
			}

			job, err := buildjob.New(cfg)
			if err != nil {
				panic(err)
			}
			defer job.Cleanup()

			if err := job.Run(); err != nil {
				panic(err)
			}
		},
	}
)

func init() {
	rootCmd.Flags().SortFlags = false

	buildCmd.Flags().StringVar(&resourceName, "resource", "", "Name of the ContainerImageBuild resource to process")
	buildCmd.Flags().StringVar(&resourceNamespace, "resource-namespace", "", "Name of the namespace containing the ContainerImageBuild resource")

	buildCmd.MarkFlagRequired("resource")
	buildCmd.MarkFlagRequired("resource-namespace")

	rootCmd.AddCommand(buildCmd)
}
