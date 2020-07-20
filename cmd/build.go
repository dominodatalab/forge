package cmd

import (
	"github.com/spf13/cobra"

	"github.com/dominodatalab/forge/internal/config"
	"github.com/dominodatalab/forge/internal/steve"
)

var (
	resourceName string

	buildCmd = &cobra.Command{
		Use:   "build",
		Short: "Launch a single OCI image build",
		Long:  "fill in the details",
		Run: func(cmd *cobra.Command, args []string) {
			opts := &config.BuildOptions{}
			if err := steve.GoBuildSomething(resourceName, opts); err != nil {
				panic(err)
			}
		},
	}
)

func init() {
	buildCmd.Flags().StringVar(&resourceName, "resource", "", "Name of the ContainerImageBuild resource to process")
	buildCmd.MarkFlagRequired("resource")

	rootCmd.AddCommand(buildCmd)
}
