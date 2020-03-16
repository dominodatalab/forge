package controllers

import (
	"context"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	forgev1alpha1 "github.com/dominodatalab/forge/api/v1alpha1"
	"github.com/dominodatalab/forge/pkg/container"
	"github.com/dominodatalab/forge/pkg/container/config"
)

// ContainerImageBuildReconciler reconciles a ContainerImageBuild object
type ContainerImageBuildReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=forge.dominodatalab.com,resources=containerimagebuilds,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=forge.dominodatalab.com,resources=containerimagebuilds/status,verbs=get;update;patch

func (r *ContainerImageBuildReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	log := r.Log.WithValues("containerimagebuild", req.NamespacedName)

	cim := forgev1alpha1.ContainerImageBuild{}
	if err := r.Get(ctx, req.NamespacedName, &cim); err != nil {
		log.Error(err, "failed to get resource")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	_, _ = r.dispatchContainerBuild(ctx, cim.Spec)

	return ctrl.Result{}, nil
}

func (r *ContainerImageBuildReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&forgev1alpha1.ContainerImageBuild{}).
		Complete(r)
}

func (r *ContainerImageBuildReconciler) dispatchContainerBuild(ctx context.Context, spec forgev1alpha1.ContainerImageBuildSpec) (string, error) {
	opts := config.BuildOptions{
		ImageName:        spec.ImageName,
		Context:          spec.Context,
		Dockerfile:       spec.Dockerfile,
		RegistryURL:      spec.PushRegistry,
		InsecureRegistry: true,
		NoCache:          spec.NoCache,
		Labels:           spec.Labels,
		BuildArgs:        spec.BuildArgs,
		CpuQuota:         spec.CpuQuota,
		Memory:           spec.Memory,
	}

	containerBuilder, err := container.NewBuilder()
	if err != nil {
		return "", err
	}
	return containerBuilder.Build(ctx, opts)
}
