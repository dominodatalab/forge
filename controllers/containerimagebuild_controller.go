package controllers

import (
	"context"
	"sort"
	"time"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	forgev1alpha1 "github.com/dominodatalab/forge/api/v1alpha1"
	"github.com/dominodatalab/forge/internal/message"
)

// blocks until all resources belonging to a ContainerImageBuild have been deleted
const gcDeleteOpt = client.PropagationPolicy(metav1.DeletePropagationForeground)

type BuildJobConfig struct {
	Image                      string
	CAImage                    string
	CustomCASecret             string
	PreparerPluginPath         string
	Labels                     map[string]string
	Annotations                map[string]string
	GrantFullPrivilege         bool
	EnableLayerCaching         bool
	PodSecurityPolicy          string
	SecurityContextConstraints string
	BrokerOpts                 *message.Options
	Volumes                    []corev1.Volume
	VolumeMounts               []corev1.VolumeMount
	EnvVar                     []corev1.EnvVar
}

type ControllerConfig struct {
	Debug                bool
	Namespace            string
	MetricsAddr          string
	EnableLeaderElection bool
	GCMaxRetentionCount  int
	GCInterval           time.Duration

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

	log.Info("Reconciling build job", "Name", build.Name, "Namespace", build.Namespace)

	if err := r.checkPrerequisites(ctx, build); err != nil {
		log.Error(err, "Failed to create job prerequisites", "Name", build.Name, "Namespace", build.Namespace)
		return ctrl.Result{}, err
	}

	if err := r.createJobForBuild(ctx, build); err != nil {
		log.Error(err, "Failed to create job", "Name", build.Name, "Namespace", build.Namespace)
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// RunGC will delete ContainerImageBuild resources that are in a "completed" or "failed" state. The oldest resources
// will be deleted first and the retentionCount will preserve N of resources for inspection.
func (r *ContainerImageBuildReconciler) RunGC(retentionCount int) {
	ctx := context.Background()
	log := r.Log.WithName("GC")
	log.Info("Launching cleanup operation")

	list := &forgev1alpha1.ContainerImageBuildList{}
	if err := r.List(ctx, list); err != nil {
		log.Error(err, "Failed to list build resources, something may be wrong")
		return
	}

	listLen := len(list.Items)
	if listLen == 0 {
		log.V(1).Info("No build resources found, aborting")
		return
	}
	log.Info("Fetched all build resources", "count", listLen)

	log.V(1).Info("Filtering builds by state", "states", []forgev1alpha1.BuildState{
		forgev1alpha1.BuildStateCompleted, forgev1alpha1.BuildStateFailed,
	})
	var builds []forgev1alpha1.ContainerImageBuild
	for _, cib := range list.Items {
		state := cib.Status.State
		if state == forgev1alpha1.BuildStateCompleted || state == forgev1alpha1.BuildStateFailed {
			builds = append(builds, cib)
		}
	}

	if len(builds) <= retentionCount {
		log.Info("Total resources are less than or equal to retention limit, aborting", "resourceCount", len(builds), "retentionCount", retentionCount)
		return
	}
	log.Info("Total resources eligible for deletion", "count", len(builds))

	sort.Slice(builds, func(i, j int) bool {
		return builds[i].CreationTimestamp.Before(&builds[j].CreationTimestamp)
	})

	for _, build := range builds[:len(builds)-retentionCount] {
		if err := r.Delete(ctx, &build, gcDeleteOpt); err != nil {
			log.Error(err, "Failed to delete build", "name", build.Name, "namespace", build.Namespace)
		}
		log.Info("Deleted build", "name", build.Name, "namespace", build.Namespace)
	}
	log.Info("Cleanup complete")
}
