package controllers

import (
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"syscall"

	"github.com/opencontainers/runc/libcontainer/system"
	"go.uber.org/zap"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	ctrl "sigs.k8s.io/controller-runtime"
	ctrlzap "sigs.k8s.io/controller-runtime/pkg/log/zap"

	forgev1alpha1 "github.com/dominodatalab/forge/api/v1alpha1"
	"github.com/dominodatalab/forge/internal/builder"
	"github.com/dominodatalab/forge/internal/message"
	_ "github.com/dominodatalab/forge/internal/unshare"
	"github.com/dominodatalab/forge/plugins/preparer"
	// +kubebuilder:scaffold:imports
)

const leaderElectionID = "forge-leader-election"

var (
	newScheme = runtime.NewScheme()
	setupLog  = ctrl.Log.WithName("setup")
)

func StartManager(namespace string, metricsAddr string, enableLeaderElection bool, brokerOpts *message.Options, preparerPluginsPath string, enableLayerCaching bool, debug bool) {
	reexec()

	atom := zap.NewAtomicLevel()
	if debug {
		atom.SetLevel(zap.DebugLevel)
	}

	logger := ctrlzap.New(func(opts *ctrlzap.Options) {
		opts.Level = &atom
	})

	ctrl.SetLogger(logger)

	preparerPlugins, err := preparer.LoadPlugins(preparerPluginsPath)
	if err != nil {
		setupLog.Error(err, fmt.Sprintf("Unable to load preparer plugins path %q", preparerPluginsPath))
		os.Exit(1)
	}

	defer func() {
		for _, preparerPlugin := range preparerPlugins {
			preparerPlugin.Kill()
		}
	}()

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:             newScheme,
		MetricsBindAddress: metricsAddr,
		LeaderElection:     enableLeaderElection,
		LeaderElectionID:   leaderElectionID,
		Port:               9443,
		Namespace:          namespace,
	})
	if err != nil {
		setupLog.Error(err, "Unable to start manager")
		os.Exit(1)
	}

	setupLog.Info("Initializing OCI builder")
	bldr, err := builder.New(preparerPlugins, enableLayerCaching, logger)
	if err != nil {
		setupLog.Error(err, "Image builder initialization failed")
		os.Exit(1)
	}

	var publisher message.Producer
	if brokerOpts != nil {
		setupLog.Info("Initializing message publisher")

		if publisher, err = message.NewProducer(brokerOpts); err != nil {
			setupLog.Error(err, "Message publisher initialization failed")
			os.Exit(1)
		}
		defer publisher.Close()
	}

	// Create a globally-scoped Kubernetes client. mgr.GetClient() can only
	// refer to resources in the same-namespace when one is provided.
	cfg, err := ctrl.GetConfig()
	if err != nil {
		setupLog.Error(err, "Could not initialize in-cluster Kubernetes config")
		os.Exit(1)
	}
	clientset, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		setupLog.Error(err, "Could not initialize Kubernetes client")
		os.Exit(1)
	}

	if err = (&ContainerImageBuildReconciler{
		Log:       ctrl.Log.WithName("controllers").WithName("ContainerImageBuild"),
		Client:    mgr.GetClient(),
		Scheme:    mgr.GetScheme(),
		Recorder:  mgr.GetEventRecorderFor("containerimagebuild-controller"),
		Builder:   bldr,
		Producer:  publisher,
		Clientset: clientset,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "Unable to create controller", "controller", "ContainerImageBuild")
		os.Exit(1)
	}
	// +kubebuilder:scaffold:builder

	setupLog.Info("Starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "Problem running manager")
		os.Exit(1)
	}
}

func reexec() {
	if len(os.Getenv("FORGE_RUNNING_TESTS")) <= 0 && len(os.Getenv("FORGE_DO_UNSHARE")) <= 0 && system.GetParentNSeuid() != 0 {
		setupLog.Info("Preparing to unshare process namespace")

		var (
			pgid int
			err  error
		)

		// On ^C, or SIGTERM handle exit.
		c := make(chan os.Signal)
		signal.Notify(c, os.Interrupt, syscall.SIGTERM)
		go func() {
			for sig := range c {
				setupLog.Info(fmt.Sprintf("Received %s, exiting.", sig.String()))
				if err := syscall.Kill(-pgid, syscall.SIGKILL); err != nil {
					setupLog.Error(err, fmt.Sprintf("syscall.Kill %d error: %v", pgid, err))
					os.Exit(1)
				}
				os.Exit(0)
			}
		}()

		// If newuidmap is not present re-exec will fail
		if _, err := exec.LookPath("newuidmap"); err != nil {
			setupLog.Error(err, fmt.Sprintf("newuidmap not found (install uidmap package?): %v", err))
		}

		// Initialize and re-exec with our unshare.
		cmd := exec.Command("/proc/self/exe", os.Args[1:]...)
		cmd.Env = append(os.Environ(), "FORGE_DO_UNSHARE=1")
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.SysProcAttr = &syscall.SysProcAttr{
			Setpgid: true,
		}
		if err := cmd.Start(); err != nil {
			setupLog.Error(err, fmt.Sprintf("cmd.Start error: %v", err))
			os.Exit(1)
		}

		pgid, err = syscall.Getpgid(cmd.Process.Pid)
		if err != nil {
			setupLog.Error(err, fmt.Sprintf("getpgid error: %v", err))
			os.Exit(1)
		}

		var (
			ws       syscall.WaitStatus
			exitCode int
		)
		for {
			// Store the exitCode before calling wait so we get the real one.
			exitCode = ws.ExitStatus()
			_, err := syscall.Wait4(-pgid, &ws, syscall.WNOHANG, nil)
			if err != nil {
				if err.Error() == "no child processes" {
					// We exited. We need to pass the correct error code from
					// the child.
					os.Exit(exitCode)
				}

				setupLog.Error(err, fmt.Sprintf("wait4 error: %v", err))
				os.Exit(1)
			}
		}
	}
}

func init() {
	_ = clientgoscheme.AddToScheme(newScheme)
	_ = forgev1alpha1.AddToScheme(newScheme)
	// +kubebuilder:scaffold:scheme
}
