package controllers

import (
	"context"
	"fmt"
	"time"

	"github.com/go-logr/logr"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	forgev1alpha1 "github.com/dominodatalab/forge/api/v1alpha1"
	"github.com/dominodatalab/forge/pkg/container"
	"github.com/dominodatalab/forge/pkg/container/config"
)

// ContainerImageBuildReconciler reconciles a ContainerImageBuild object
type ContainerImageBuildReconciler struct {
	client.Client
	Log      logr.Logger
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder
	Builder  container.RuntimeBuilder
}

// +kubebuilder:rbac:groups=forge.dominodatalab.com,resources=containerimagebuilds,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=forge.dominodatalab.com,resources=containerimagebuilds/status,verbs=get;update;patch

func (r *ContainerImageBuildReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	log := r.Log.WithValues("containerimagebuild", req.Name)

	var result ctrl.Result
	var build forgev1alpha1.ContainerImageBuild

	// attempt to load resource by name and ignore not-found errors
	if err := r.Get(ctx, req.NamespacedName, &build); err != nil {
		log.Error(err, "Unable to find resource")
		return result, client.IgnoreNotFound(err)
	}

	// ignore resources that have been processed on start
	if len(build.Status.State) != 0 {
		log.Info("Skipping resource", "state", build.Status.State)
		return result, nil
	}

	// mark resource status and update before launching build
	build.Status.State = forgev1alpha1.Building
	build.Status.BuildStartedAt = &metav1.Time{Time: time.Now()}

	if err := r.updateResourceStatus(ctx, log, &build); err != nil {
		return result, err
	}

	// construct build directives and dispatch operation
	opts := config.BuildOptions{
		ImageName:        build.Spec.ImageName,
		Context:          build.Spec.Context,
		RegistryURL:      build.Spec.PushRegistry,
		NoCache:          build.Spec.NoCache,
		Labels:           build.Spec.Labels,
		BuildArgs:        build.Spec.BuildArgs,
		CpuQuota:         build.Spec.CpuQuota,
		Memory:           build.Spec.Memory,
		InsecureRegistry: true,
	}

	imageURL, err := r.Builder.Build(ctx, opts)
	if err != nil {
		log.Error(err, "Build process failed")

		build.Status.State = forgev1alpha1.Failed
		build.Status.ErrorMessage = err.Error()

		if uErr := r.updateResourceStatus(ctx, log, &build); uErr != nil {
			err = fmt.Errorf("multiple failures occurred: %w: followed by %v", err, uErr)
			return result, err
		}

		return result, nil
	}

	// mark resource status to indicate build was successful
	build.Status.ImageURL = imageURL
	build.Status.State = forgev1alpha1.Completed
	build.Status.BuildCompletedAt = &metav1.Time{Time: time.Now()}

	if err := r.updateResourceStatus(ctx, log, &build); err != nil {
		return result, err
	}

	// reconcile result will ensure this event is not enqueued again
	return result, nil
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

func (r *ContainerImageBuildReconciler) updateResourceStatus(ctx context.Context, log logr.Logger, build *forgev1alpha1.ContainerImageBuild) error {
	err := r.Status().Update(ctx, build)
	if err != nil {
		log.Error(err, "Unable to update status")

		msg := fmt.Sprintf("Forge was unable to update this resource status: %v", err)
		r.Recorder.Event(build, "Warning", "UpdateFailed", msg)
	}

	return err
}
