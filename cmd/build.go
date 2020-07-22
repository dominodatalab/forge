package cmd

import (
	"github.com/spf13/cobra"

	"github.com/dominodatalab/forge/internal/buildjob"
)

var (
	resourceName string

	buildCmd = &cobra.Command{
		Use:   "build",
		Short: "Launch a single OCI image build",
		Long:  "fill in the details",
		Run: func(cmd *cobra.Command, args []string) {
			cfg := buildjob.Config{
				ResourceName:        resourceName,
				BrokerOpts:          brokerOpts,
				PreparerPluginsPath: preparerPluginsPath,
				EnableLayerCaching:  enableLayerCaching,
				Debug:               debug,
			}

			job, err := buildjob.New(cfg)
			if err != nil {
				panic(err)
			}

			if err := job.Run(); err != nil {
				panic(err)
			}
			job.Cleanup()
		},
	}
)

func init() {
	rootCmd.Flags().SortFlags = false

	buildCmd.Flags().StringVar(&resourceName, "resource", "", "Name of the ContainerImageBuild resource to process")
	if err := buildCmd.MarkFlagRequired("resource"); err != nil {
		panic(err)
	}

	rootCmd.AddCommand(buildCmd)
}
