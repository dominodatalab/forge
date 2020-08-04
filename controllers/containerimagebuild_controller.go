package controllers

import (
	"context"

	"github.com/go-logr/logr"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	forgev1alpha1 "github.com/dominodatalab/forge/api/v1alpha1"
	"github.com/dominodatalab/forge/internal/message"
)

type BuildJobConfig struct {
	Image              string
	CAImage            string
	CustomCASecret     string
	PreparerPluginPath string
	Labels             map[string]string
	Annotations        map[string]string
	GrantFullPrivilege bool
	EnableLayerCaching bool
	BrokerOpts         *message.Options
	Volumes            []corev1.Volume
	VolumeMounts       []corev1.VolumeMount
	EnvVar             []corev1.EnvVar
}

type ControllerConfig struct {
	Debug                bool
	Namespace            string
	MetricsAddr          string
	EnableLeaderElection bool

	JobConfig *BuildJobConfig
}

// ContainerImageBuildReconciler reconciles a ContainerImageBuild object
type ContainerImageBuildReconciler struct {
	client.Client
	*kubernetes.Clientset
	Log      logr.Logger
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder

	JobConfig *BuildJobConfig
}

func (r *ContainerImageBuildReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&forgev1alpha1.ContainerImageBuild{}).
		WithEventFilter(predicate.Funcs{ // NOTE this ignores update/delete events
			CreateFunc: func(event event.CreateEvent) bool {
				return true
			},
			UpdateFunc: func(updateEvent event.UpdateEvent) bool {
				return false
			},
			DeleteFunc: func(deleteEvent event.DeleteEvent) bool {
				return false
			},
		}).
		Complete(r)
}

// +kubebuilder:rbac:groups=forge.dominodatalab.com,resources=containerimagebuilds,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=forge.dominodatalab.com,resources=containerimagebuilds/status,verbs=get;update;patch

func (r *ContainerImageBuildReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	log := r.Log.WithValues("containerimagebuild", req.NamespacedName)

	// attempt to load resource by name and ignore not-found errors
	build := &forgev1alpha1.ContainerImageBuild{}
	if err := r.Get(ctx, req.NamespacedName, build); err != nil {
		log.Error(err, "Unable to find resource")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	existing := &batchv1.Job{}
	err := r.Get(ctx, types.NamespacedName{Name: build.Name, Namespace: build.Namespace}, existing)
	if err != nil {
		// requeue when get job fails
		if !apierrors.IsNotFound(err) {
			log.Error(err, "Failed to get build job")
			return ctrl.Result{}, err
		}

		if err := r.checkPrerequisites(ctx, build); err != nil {
			log.Error(err, "Failed to create job prerequisites", "Name", build.Name, "Namespace", build.Namespace)
			return ctrl.Result{}, err
		}

		job, err := r.jobForBuild(build)
		if err != nil {
			log.Error(err, "Failed to create job", "Name", build.Name, "Namespace", build.Namespace)
			return ctrl.Result{}, err
		}

		log.Info("Creating new build job", "Name", build.Name, "Namespace", build.Namespace)
		if err := r.Create(ctx, job); err != nil {
			// requeue when create job fails
			log.Error(err, "Failed to create build job", "Name", build.Name, "Namespace", build.Namespace)
			return ctrl.Result{}, err
		}
	}

	// TODO: add a back reference on CIB to the build pod

	return ctrl.Result{}, nil
}
