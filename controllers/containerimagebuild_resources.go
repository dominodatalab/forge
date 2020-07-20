package controllers

import (
	"context"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	forgev1alpha1 "github.com/dominodatalab/forge/api/v1alpha1"
)

func (r *ContainerImageBuildReconciler) checkPrerequisites(ctx context.Context, cib *forgev1alpha1.ContainerImageBuild) error {
	if err := r.checkServiceAccount(ctx, cib); err != nil {
		return err
	}
	if err := r.checkRole(ctx, cib); err != nil {
		return err
	}
	if err := r.checkRoleBinding(ctx, cib); err != nil {
		return err
	}

	return nil
}

func (r *ContainerImageBuildReconciler) checkServiceAccount(ctx context.Context, cib *forgev1alpha1.ContainerImageBuild) error {
	err := r.Get(ctx, types.NamespacedName{Name: cib.Name, Namespace: cib.Namespace}, &corev1.ServiceAccount{})
	if err == nil {
		return nil
	}
	if !apierrors.IsNotFound(err) {
		return err
	}

	sa := &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cib.Name,
			Namespace: cib.Namespace,
			Labels:    cib.Labels,
		},
	}
	if err := controllerutil.SetControllerReference(cib, sa, r.Scheme); err != nil {
		return err
	}

	return r.Create(ctx, sa)
}

func (r *ContainerImageBuildReconciler) checkRole(ctx context.Context, cib *forgev1alpha1.ContainerImageBuild) error {
	err := r.Get(ctx, types.NamespacedName{Name: cib.Name, Namespace: cib.Namespace}, &rbacv1.Role{})
	if err == nil {
		return nil
	}
	if !apierrors.IsNotFound(err) {
		return err
	}

	role := &rbacv1.Role{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cib.Name,
			Namespace: cib.Namespace,
			Labels:    cib.Labels,
		},
		Rules: []rbacv1.PolicyRule{
			{
				APIGroups: []string{forgev1alpha1.GroupVersion.Group},
				Resources: []string{"containerimagebuilds"},
				Verbs:     []string{"get"},
			},
			{
				APIGroups: []string{forgev1alpha1.GroupVersion.Group},
				Resources: []string{"containerimagebuilds/status"},
				Verbs:     []string{"update"},
			},
		},
	}
	if err := controllerutil.SetControllerReference(cib, role, r.Scheme); err != nil {
		return err
	}

	return r.Create(ctx, role)
}

func (r *ContainerImageBuildReconciler) checkRoleBinding(ctx context.Context, cib *forgev1alpha1.ContainerImageBuild) error {
	err := r.Get(ctx, types.NamespacedName{Name: cib.Name, Namespace: cib.Namespace}, &rbacv1.RoleBinding{})
	if err == nil {
		return nil
	}
	if !apierrors.IsNotFound(err) {
		return err
	}

	binding := &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cib.Name,
			Namespace: cib.Namespace,
			Labels:    cib.Labels,
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: rbacv1.GroupName,
			Kind:     "Role",
			Name:     cib.Name,
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      rbacv1.ServiceAccountKind,
				Name:      cib.Name,
				Namespace: cib.Namespace,
			},
		},
	}
	if err := controllerutil.SetControllerReference(cib, binding, r.Scheme); err != nil {
		return err
	}

	return r.Create(ctx, binding)
}

func (r *ContainerImageBuildReconciler) jobForBuild(cib *forgev1alpha1.ContainerImageBuild) *batchv1.Job {
	commonMeta := metav1.ObjectMeta{
		Name:      cib.Name,
		Namespace: cib.Namespace,
		Labels:    cib.Labels,
	}
	job := &batchv1.Job{
		ObjectMeta: commonMeta,
		Spec: batchv1.JobSpec{

			BackoffLimit:            pointer.Int32Ptr(0),
			ActiveDeadlineSeconds:   pointer.Int64Ptr(3600),
			TTLSecondsAfterFinished: pointer.Int32Ptr(0),
			Template: corev1.PodTemplateSpec{
				ObjectMeta: commonMeta,
				Spec: corev1.PodSpec{
					ServiceAccountName: cib.Name,
					RestartPolicy:      corev1.RestartPolicyNever,
					Containers: []corev1.Container{
						{
							Name:  "forge-build",
							Image: r.BuildJobImage,
							Args:  r.prepareJobArgs(cib),
						},
					},
				},
			},
		},
	}

	controllerutil.SetControllerReference(cib, job, r.Scheme)
	return job
}

func (r *ContainerImageBuildReconciler) prepareJobArgs(cib *forgev1alpha1.ContainerImageBuild) []string {
	args := []string{
		"build",
		"--resource",
		cib.Name,
	}

	return args
}
