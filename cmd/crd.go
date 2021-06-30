package cmd

import (
	"context"

	"github.com/spf13/cobra"

	"github.com/dominodatalab/forge/internal/crd"
)

var (
	applyCrdCmd = &cobra.Command{
		Use:   "crd-apply",
		Short: "Apply the ContainerImageBuild CRD to a cluster",
		RunE: func(cmd *cobra.Command, args []string) error {
			return crd.Apply(context.TODO())
		},
	}

	deleteCrdCmd = &cobra.Command{
		Use:   "crd-delete",
		Short: "Remove the ContainerImageBuild CRD from a cluster",
		RunE: func(cmd *cobra.Command, args []string) error {
			return crd.Delete(context.TODO())
		},
	}
)

func init() {
	rootCmd.AddCommand(applyCrdCmd)
	rootCmd.AddCommand(deleteCrdCmd)
}
