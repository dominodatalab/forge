package buildjob

import (
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"syscall"

	"github.com/go-logr/logr"
	"github.com/opencontainers/runc/libcontainer/system"
	"github.com/pkg/errors"
	"go.uber.org/zap"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	ctrlzap "sigs.k8s.io/controller-runtime/pkg/log/zap"
)

func NewLogger() logr.Logger {
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

func reexec(log logr.Logger) {
	if len(os.Getenv("FORGE_RUNNING_TESTS")) <= 0 && len(os.Getenv("FORGE_DO_UNSHARE")) <= 0 && system.GetParentNSeuid() != 0 {
		log.Info("Preparing to unshare process namespace")

		var (
			pgid int
			err  error
		)

		// On ^C, or SIGTERM handle exit.
		c := make(chan os.Signal)
		signal.Notify(c, os.Interrupt, syscall.SIGTERM)
		go func() {
			for sig := range c {
				log.Info(fmt.Sprintf("Received %s, exiting.", sig.String()))
				if err := syscall.Kill(-pgid, syscall.SIGKILL); err != nil {
					log.Error(err, fmt.Sprintf("syscall.Kill %d error: %v", pgid, err))
					os.Exit(1)
				}
				os.Exit(0)
			}
		}()

		// If newuidmap is not present re-exec will fail
		if _, err := exec.LookPath("newuidmap"); err != nil {
			log.Error(err, fmt.Sprintf("newuidmap not found (install uidmap package?): %v", err))
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
			log.Error(err, fmt.Sprintf("cmd.Start error: %v", err))
			os.Exit(1)
		}

		pgid, err = syscall.Getpgid(cmd.Process.Pid)
		if err != nil {
			log.Error(err, fmt.Sprintf("getpgid error: %v", err))
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

				log.Error(err, fmt.Sprintf("wait4 error: %v", err))
				os.Exit(1)
			}
		}
	}
}
