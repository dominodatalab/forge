package controllers

import (
	"context"
	"fmt"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	policyv1beta1 "k8s.io/api/policy/v1beta1"
	rbacv1 "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	forgev1alpha1 "github.com/dominodatalab/forge/api/v1alpha1"
)

// creates all supporting resources required by build job
func (r *ContainerImageBuildReconciler) checkPrerequisites(ctx context.Context, cib *forgev1alpha1.ContainerImageBuild) error {
	if err := r.checkServiceAccount(ctx, cib); err != nil {
		return err
	}
	if err := r.checkPodSecurityPolicy(ctx, cib); err != nil {
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

// creates build job service account when missing
func (r *ContainerImageBuildReconciler) checkServiceAccount(ctx context.Context, cib *forgev1alpha1.ContainerImageBuild) error {
	return r.withOwnedResource(ctx, cib, &corev1.ServiceAccount{}, func() interface{} {
		return &corev1.ServiceAccount{
			ObjectMeta: metav1.ObjectMeta{
				Name:      cib.Name,
				Namespace: cib.Namespace,
				Labels:    cib.Labels,
			},
		}
	})
}

// creates build job pod security policy when missing
func (r *ContainerImageBuildReconciler) checkPodSecurityPolicy(ctx context.Context, cib *forgev1alpha1.ContainerImageBuild) error {
	return r.withOwnedResource(ctx, cib, &policyv1beta1.PodSecurityPolicy{}, func() interface{} {
		psp := &policyv1beta1.PodSecurityPolicy{
			ObjectMeta: metav1.ObjectMeta{
				Name:      cib.Name,
				Namespace: cib.Namespace,
				Labels:    cib.Labels,
				Annotations: map[string]string{
					"seccomp.security.alpha.kubernetes.io/allowedProfileNames": "unconfined,runtime/default",
					"apparmor.security.beta.kubernetes.io/allowedProfileNames": "unconfined,runtime/default",
					"seccomp.security.alpha.kubernetes.io/defaultProfileName":  "unconfined",
					"apparmor.security.beta.kubernetes.io/defaultProfileName":  "unconfined",
				},
			},
			Spec: policyv1beta1.PodSecurityPolicySpec{
				AllowPrivilegeEscalation: pointer.BoolPtr(true),
				AllowedProcMountTypes: []corev1.ProcMountType{
					corev1.DefaultProcMount,
					corev1.UnmaskedProcMount,
				},
				Privileged:  false,
				HostPID:     false,
				HostIPC:     false,
				HostNetwork: false,
				RunAsUser: policyv1beta1.RunAsUserStrategyOptions{
					Rule: policyv1beta1.RunAsUserStrategyMustRunAsNonRoot,
				},
				FSGroup: policyv1beta1.FSGroupStrategyOptions{
					Rule: policyv1beta1.FSGroupStrategyMustRunAs,
					Ranges: []policyv1beta1.IDRange{
						{
							Min: 1,
							Max: 65535,
						},
					},
				},
				SupplementalGroups: policyv1beta1.SupplementalGroupsStrategyOptions{
					Rule: policyv1beta1.SupplementalGroupsStrategyMustRunAs,
					Ranges: []policyv1beta1.IDRange{
						{
							Min: 1,
							Max: 65535,
						},
					},
				},
				SELinux: policyv1beta1.SELinuxStrategyOptions{
					Rule: policyv1beta1.SELinuxStrategyRunAsAny,
				},
				Volumes: []policyv1beta1.FSType{
					policyv1beta1.EmptyDir,
					policyv1beta1.PersistentVolumeClaim,
					policyv1beta1.Secret,
				},
			},
		}

		if r.JobConfig.GrantFullPrivilege {
			psp.Spec.Privileged = true
			psp.Spec.RunAsUser = policyv1beta1.RunAsUserStrategyOptions{
				Rule: policyv1beta1.RunAsUserStrategyRunAsAny,
			}
		}

		return psp
	})
}

// creates build role when missing
func (r *ContainerImageBuildReconciler) checkRole(ctx context.Context, cib *forgev1alpha1.ContainerImageBuild) error {
	return r.withOwnedResource(ctx, cib, &rbacv1.Role{}, func() interface{} {
		return &rbacv1.Role{
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
				{
					APIGroups: []string{""},
					Resources: []string{"secrets"},
					Verbs:     []string{"get"},
				},
				{
					APIGroups:     []string{"policy"},
					Resources:     []string{"podsecuritypolicies"},
					ResourceNames: []string{cib.Name},
					Verbs:         []string{"use"},
				},
			},
		}
	})
}

// creates build job service role binding when missing
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

// generates build job definition using container image build spec
func (r *ContainerImageBuildReconciler) jobForBuild(cib *forgev1alpha1.ContainerImageBuild) (*batchv1.Job, error) {
	// setup pod metadata
	podMeta := metav1.ObjectMeta{
		Name:      cib.Name,
		Namespace: cib.Namespace,
		Labels:    cib.Labels,
		Annotations: map[string]string{
			"container.apparmor.security.beta.kubernetes.io/forge-build": "unconfined",
			"container.seccomp.security.alpha.kubernetes.io/forge-build": "unconfined",
		},
	}
	if podMeta.Labels == nil {
		podMeta.Labels = make(map[string]string)
	}
	for k, v := range r.JobConfig.Labels {
		podMeta.Labels[k] = v
	}
	for k, v := range r.JobConfig.Annotations {
		podMeta.Annotations[k] = v
	}

	// setup pod environment variables
	env := []corev1.EnvVar{
		// TODO: remove once forge includes the following commit:
		// 	https://github.com/moby/buildkit/commit/ec5d112053221b41602536cdaa6cc958d7183e2b
		{
			Name:  "PROGRESS_NO_TRUNC",
			Value: "1",
		},
	}
	for _, ev := range r.JobConfig.EnvVar {
		env = append(env, ev)
	}

	// setup security context
	secCtx := &corev1.SecurityContext{
		RunAsUser: pointer.Int64Ptr(1000),
	}
	if r.JobConfig.GrantFullPrivilege {
		secCtx.RunAsUser = pointer.Int64Ptr(0)
		secCtx.Privileged = pointer.BoolPtr(true)
	}

	// setup volumes and mounts used by main container
	var volumes []corev1.Volume
	for _, volume := range r.JobConfig.Volumes {
		volumes = append(volumes, volume)
	}

	var volumeMounts []corev1.VolumeMount
	for _, mount := range r.JobConfig.VolumeMounts {
		volumeMounts = append(volumeMounts, mount)
	}

	// optionally configure the custom CA init container w/ additional volumes/mounts
	var initContainers []corev1.Container
	if r.JobConfig.CustomCASecret != "" {
		caTLSVol := corev1.Volume{
			Name: "ca-tls",
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: r.JobConfig.CustomCASecret,
				},
			},
		}
		sslCertsVol := corev1.Volume{
			Name: "ssl-certs",
			VolumeSource: corev1.VolumeSource{
				EmptyDir: &corev1.EmptyDirVolumeSource{},
			},
		}
		volumes = append(volumes, caTLSVol, sslCertsVol)

		caTLSVolMount := corev1.VolumeMount{
			Name:      caTLSVol.Name,
			MountPath: "/tmp/forge/ca-tls",
		}
		sslCertsVolMount := corev1.VolumeMount{
			Name:      sslCertsVol.Name,
			MountPath: "/etc/ssl/certs",
		}

		initContainers = append(initContainers, corev1.Container{
			Name:  "init-ca-certs",
			Image: r.JobConfig.CAImage,
			Env: []corev1.EnvVar{
				{
					Name:  "CERT_DIR",
					Value: "/tmp/forge/ca-tls",
				},
			},
			VolumeMounts: []corev1.VolumeMount{caTLSVolMount, sslCertsVolMount},
		})

		forgeBuildMount := sslCertsVolMount.DeepCopy()
		forgeBuildMount.ReadOnly = true
		volumeMounts = append(volumeMounts, *forgeBuildMount)
	}

	// construct final job object
	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cib.Name,
			Namespace: cib.Namespace,
			Labels:    cib.Labels,
		},
		Spec: batchv1.JobSpec{
			BackoffLimit:            pointer.Int32Ptr(0),
			ActiveDeadlineSeconds:   pointer.Int64Ptr(3600),
			TTLSecondsAfterFinished: pointer.Int32Ptr(0),
			Template: corev1.PodTemplateSpec{
				ObjectMeta: podMeta,
				Spec: corev1.PodSpec{
					ServiceAccountName: cib.Name,
					RestartPolicy:      corev1.RestartPolicyNever,
					InitContainers:     initContainers,
					Containers: []corev1.Container{
						{
							Name:            "forge-build",
							Image:           r.JobConfig.Image,
							Args:            r.prepareJobArgs(cib),
							Env:             env,
							SecurityContext: secCtx,
							VolumeMounts:    volumeMounts,
						},
					},
					Volumes: volumes,
				},
			},
		},
	}

	err := controllerutil.SetControllerReference(cib, job, r.Scheme)
	return job, err
}

// builds cli args required to launch forge in "build mode" inside a job
func (r *ContainerImageBuildReconciler) prepareJobArgs(cib *forgev1alpha1.ContainerImageBuild) []string {
	args := []string{
		"build",
		fmt.Sprintf("--resource=%s", cib.Name),
		fmt.Sprintf("--enable-layer-caching=%t", r.JobConfig.EnableLayerCaching),
		fmt.Sprintf("--preparer-plugins-path=%s", r.JobConfig.PreparerPluginPath),
	}

	if r.JobConfig.BrokerOpts != nil {
		opts := r.JobConfig.BrokerOpts
		bs := []string{
			fmt.Sprintf("--broker=%s", opts.Broker),
			fmt.Sprintf("--amqp-queue=%s", opts.AmqpQueue),
			fmt.Sprintf("--amqp-uri=%s", opts.AmqpURI),
		}
		args = append(args, bs...)
	}

	return args
}

// checks if runtime object exists. if it does not exist, ownership is assigned to a container image build resource and
// the object is then created. otherwise, this procedure results in a no-op.
func (r *ContainerImageBuildReconciler) withOwnedResource(ctx context.Context, cib *forgev1alpha1.ContainerImageBuild, target runtime.Object, fn func() interface{}) error {
	err := r.Get(ctx, types.NamespacedName{Name: cib.Name, Namespace: cib.Namespace}, target)
	if err == nil {
		return nil
	}
	if !apierrors.IsNotFound(err) {
		return err
	}

	obj := fn()

	if err := controllerutil.SetControllerReference(cib, obj.(metav1.Object), r.Scheme); err != nil {
		return err
	}
	return r.Create(ctx, obj.(runtime.Object))
}
