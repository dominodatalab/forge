/*
Copyright 2020 Domino Data Lab, Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"context"
	"time"

	"github.com/go-logr/logr"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	forgev1alpha1 "github.com/dominodatalab/forge/api/v1alpha1"
	"github.com/dominodatalab/forge/pkg/config"
	"github.com/dominodatalab/forge/pkg/container"
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

	log.Info("fetching resource")
	cim := forgev1alpha1.ContainerImageBuild{}
	if err := r.Get(ctx, req.NamespacedName, &cim); err != nil {
		log.Error(err, "failed to get resource")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	if cim.Status.State != "" {
		return ctrl.Result{}, nil
	}

	cim.Status.State = forgev1alpha1.Building
	cim.Status.BuildStartedAt = &metav1.Time{Time: time.Now()}
	if err := r.Status().Update(ctx, &cim); err != nil {
		log.Error(err, "unable to update status")
		return ctrl.Result{}, err
	}

	image, err := r.dispatchContainerBuild(cim.Spec)
	if err != nil {
		log.Error(err, "container image build failed")
		return ctrl.Result{}, err
	}

	cim.Status.ImageURL = image
	cim.Status.State = forgev1alpha1.Completed
	cim.Status.BuildCompletedAt = &metav1.Time{Time: time.Now()}
	if err := r.Status().Update(ctx, &cim); err != nil {
		log.Error(err, "unable to update status")
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *ContainerImageBuildReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&forgev1alpha1.ContainerImageBuild{}).
		Complete(r)
}

func (r *ContainerImageBuildReconciler) dispatchContainerBuild(spec forgev1alpha1.ContainerImageBuildSpec) (string, error) {
	opts := config.BuildOptions{
		Image: config.Image{
			Name:     spec.Build.ImageName,
			Commands: spec.Build.Commands,
		},
		Registry: config.Registry{
			ServerURL: spec.Build.PushRegistry,
			Insecure:  true,
		},
	}

	// NOTE should we move this into a constructor?
	// 	doing so will mean that we need to choose a "single" builder for all builds
	// 	but that's probably okay
	containerBuilder, err := container.NewBuilder()
	if err != nil {
		return "", err
	}
	return containerBuilder.Build(context.TODO(), opts)
}
