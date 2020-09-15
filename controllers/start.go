package controllers

import (
	"os"
	"time"

	"github.com/go-logr/zapr"
	"github.com/newrelic/go-agent/v3/integrations/nrzap"
	"github.com/newrelic/go-agent/v3/newrelic"

	"go.uber.org/zap"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	ctrl "sigs.k8s.io/controller-runtime"
	ctrlzap "sigs.k8s.io/controller-runtime/pkg/log/zap"

	forgev1alpha1 "github.com/dominodatalab/forge/api/v1alpha1"
	// +kubebuilder:scaffold:imports
)

var (
	newScheme = runtime.NewScheme()
	setupLog  = ctrl.Log.WithName("setup")
)

const newrelicShutdownTimeout = 5 * time.Second
const leaderElectionID = "forge-leader-election"

func StartManager(cfg ControllerConfig) {
	atom := zap.NewAtomicLevel()
	if cfg.Debug {
		atom.SetLevel(zap.DebugLevel)
	}

	zapLogger := ctrlzap.NewRaw(func(opts *ctrlzap.Options) {
		opts.Level = &atom
		opts.Development = true
	})
	logger := zapr.NewLogger(zapLogger)
	ctrl.SetLogger(logger)

	newrelicApp, err := newrelic.NewApplication(
		newrelic.ConfigEnabled(false),
		nrzap.ConfigLogger(zapLogger),
		newrelic.ConfigFromEnvironment(),
	)
	if err != nil {
		setupLog.Error(err, "unable to create New Relic Application")
		os.Exit(1)
	}
	defer newrelicApp.Shutdown(newrelicShutdownTimeout)

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:             newScheme,
		MetricsBindAddress: cfg.MetricsAddr,
		LeaderElection:     cfg.EnableLeaderElection,
		LeaderElectionID:   leaderElectionID,
		Port:               9443,
		Namespace:          cfg.Namespace,
	})
	if err != nil {
		setupLog.Error(err, "Unable to start manager")
		os.Exit(1)
	}

	controller := &ContainerImageBuildReconciler{
		Log:       ctrl.Log.WithName("controllers").WithName("ContainerImageBuild"),
		Client:    mgr.GetClient(),
		Scheme:    mgr.GetScheme(),
		Recorder:  mgr.GetEventRecorderFor("containerimagebuild-controller"),
		JobConfig: cfg.JobConfig,
		NewRelic:  newrelicApp,
	}

	if err = controller.SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "Unable to create controller", "controller", "ContainerImageBuild")
		os.Exit(1)
	}
	// +kubebuilder:scaffold:builder

	if cfg.GCInterval > 0 {
		ticker := time.NewTicker(cfg.GCInterval)
		defer func() {
			setupLog.Info("Shutting down GC routine")
			ticker.Stop()
		}()

		go func() {
			for range ticker.C {
				controller.RunGC(cfg.GCMaxRetentionCount)
			}
		}()
	} else {
		setupLog.Info("Auto-GC disabled, you must delete ContainerImageBuild resources on your own")
	}

	setupLog.Info("Starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "Problem running manager")
		os.Exit(1)
	}
}

func init() {
	_ = clientgoscheme.AddToScheme(newScheme)
	_ = forgev1alpha1.AddToScheme(newScheme)
	// +kubebuilder:scaffold:scheme
}
