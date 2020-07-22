package cmd

import (
	"errors"
	"io/ioutil"
	"strings"

	"github.com/spf13/cobra"

	"github.com/dominodatalab/forge/internal/buildjob"
)

var (
	resourceName      string
	resourceNamespace string

	buildCmd = &cobra.Command{
		Use:     "build",
		Short:   "Launch a single OCI image build",
		Long:    "fill in the details",
		PreRunE: processResourceArgs,
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

func processResourceArgs(cmd *cobra.Command, args []string) error {
	var errMsgs []string

	if resourceName == "" {
		errMsgs = append(errMsgs, "'--resource' is required")
	}
	if resourceNamespace == "" {
		bs, err := ioutil.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/namespace")
		if err != nil {
			errMsgs = append(errMsgs, "'--resource-namespace' is required")
		}
		resourceNamespace = string(bs)
	}

	if errMsgs != nil {
		return errors.New(strings.Join(errMsgs, ", "))
	}
	return nil
}

func init() {
	rootCmd.Flags().SortFlags = false

	buildCmd.Flags().StringVar(&resourceName, "resource", "", "Name of the ContainerImageBuild resource to process")
	buildCmd.Flags().StringVar(&resourceNamespace, "resource-namespace", "", "Name of the namespace containing the ContainerImageBuild resource")

	rootCmd.AddCommand(buildCmd)
}
