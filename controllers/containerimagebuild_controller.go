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
	"fmt"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	forgev1alpha1 "github.com/dominodatalab/forge/api/v1alpha1"
	"github.com/dominodatalab/forge/pkg/container/runc"
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

	log.Info("fetching ContainerImageBuild resource")
	var cim forgev1alpha1.ContainerImageBuild
	if err := r.Get(ctx, req.NamespacedName, &cim); err != nil {
		log.Error(err, "failed to get ContainerImageBuild resource")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	//log.Info("updating ....")
	//cim.Status.StartedAt = &metav1.Time{Time: time.Now()}
	//if err := r.Status().Update(ctx, &cim); err != nil {
	//	log.Error(err, "unable to update status")
	//	return ctrl.Result{}, err
	//}
	//log.Info("resource status synced")

	//cim.Status.StartedAt = &metav1.Time{Time: time.Now()}
	//if err := r.Status().Update(ctx, &cim); err != nil {
	//	log.Error(err, "unable to update status")
	//	return ctrl.Result{}, err
	//}

	builder := runc.NewRuncBuilder()
	out, err := builder.Build(ctx, cim.Spec)
	if err != nil {
		log.Error(err, "container build failed")
	}
	fmt.Println(out)

	return ctrl.Result{}, nil
}

func (r *ContainerImageBuildReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&forgev1alpha1.ContainerImageBuild{}).
		Complete(r)
}
