package buildjob

import (
	"fmt"
	"strings"

	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	"go.uber.org/zap"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	ctrlzap "sigs.k8s.io/controller-runtime/pkg/log/zap"
)

func newLogger() logr.Logger {
	atom := zap.NewAtomicLevel()
	return ctrlzap.New(func(options *ctrlzap.Options) {
		options.Level = &atom
	})
}

// These are not informative errors and are captured by the progressui display in a better way
var ignoredErrors = map[string]interface{}{
	"runc did not terminate successfully": nil,
}

// This logs the underlying error from a build when the display channels inside builder.embedded have not yet been initialized
// or the error comes after the embedded driver has been run (e.g. image size limit has been hit)
func logError(log logr.Logger, err error) {
	if unwrappedError := errors.Unwrap(err); unwrappedError != nil {
		err = unwrappedError
	}

	cause := errors.Cause(err)
	if _, ok := ignoredErrors[cause.Error()]; ok {
		return
	}

	log.Info(strings.Repeat("=", 70))
	log.Info(fmt.Sprintf("Error during image build and push: %s", cause.Error()))
}

// set up standard and custom k8s clients
func loadKubernetesConfig() (*rest.Config, error) {
	kubeconfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		clientcmd.NewDefaultClientConfigLoadingRules(),
		&clientcmd.ConfigOverrides{},
	)

	if cfg, err := kubeconfig.ClientConfig(); err == nil {
		return cfg, nil
	}

	return rest.InClusterConfig()
}
