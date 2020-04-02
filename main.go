package main

import (
	"flag"
	"os"
	"os/exec"
	"os/signal"
	"syscall"

	"github.com/opencontainers/runc/libcontainer/system"
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	forgev1alpha1 "github.com/dominodatalab/forge/api/v1alpha1"
	"github.com/dominodatalab/forge/controllers"
	_ "github.com/dominodatalab/forge/internal/unshare"
	"github.com/dominodatalab/forge/pkg/container"
	// +kubebuilder:scaffold:imports
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	_ = clientgoscheme.AddToScheme(scheme)

	_ = forgev1alpha1.AddToScheme(scheme)
	// +kubebuilder:scaffold:scheme
}

func main() {
	var metricsAddr string
	var enableLeaderElection bool
	flag.StringVar(&metricsAddr, "metrics-addr", ":8080", "The address the metric endpoint binds to.")
	flag.BoolVar(&enableLeaderElection, "enable-leader-election", true,
		"Enable leader election for controller manager. Enabling this will ensure there is only one active controller manager.")
	flag.Parse()

	reexec()

	ctrl.SetLogger(zap.New(func(o *zap.Options) {
		o.Development = true
	}))

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:             scheme,
		MetricsBindAddress: metricsAddr,
		LeaderElection:     enableLeaderElection,
		Port:               9443,
	})
	if err != nil {
		setupLog.Error(err, "Unable to start manager")
		os.Exit(1)
	}

	setupLog.Info("Initializing OCI builder")
	builder, err := container.NewBuilder()
	if err != nil {
		setupLog.Error(err, "Image builder initialization failed")
		os.Exit(1)
	}

	if err = (&controllers.ContainerImageBuildReconciler{
		Log:      ctrl.Log.WithName("controllers").WithName("ContainerImageBuild"),
		Client:   mgr.GetClient(),
		Scheme:   mgr.GetScheme(),
		Recorder: mgr.GetEventRecorderFor("containerimagebuild-controller"),
		Builder:  builder,
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
	if len(os.Getenv("IMG_DO_UNSHARE")) <= 0 && system.GetParentNSeuid() != 0 {
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
				logrus.Infof("Received %s, exiting.", sig.String())
				if err := syscall.Kill(-pgid, syscall.SIGKILL); err != nil {
					logrus.Fatalf("syscall.Kill %d error: %v", pgid, err)
					continue
				}
				os.Exit(0)
			}
		}()

		// If newuidmap is not present re-exec will fail
		if _, err := exec.LookPath("newuidmap"); err != nil {
			logrus.Fatalf("newuidmap not found (install uidmap package?): %v", err)
		}

		// Initialize and re-exec with our unshare.
		cmd := exec.Command("/proc/self/exe", os.Args[1:]...)
		cmd.Env = append(os.Environ(), "IMG_DO_UNSHARE=1")
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.SysProcAttr = &syscall.SysProcAttr{
			Setpgid: true,
		}
		if err := cmd.Start(); err != nil {
			logrus.Fatalf("cmd.Start error: %v", err)
		}

		pgid, err = syscall.Getpgid(cmd.Process.Pid)
		if err != nil {
			logrus.Fatalf("getpgid error: %v", err)
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

				logrus.Fatalf("wait4 error: %v", err)
			}
		}
	}
}
