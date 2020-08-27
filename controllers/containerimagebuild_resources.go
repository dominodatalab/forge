package controllers

import (
	"context"
	"fmt"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	forgev1alpha1 "github.com/dominodatalab/forge/api/v1alpha1"
	"github.com/dominodatalab/forge/internal/config"
)

const (
	rootlesskitCommand = "rootlesskit"
	forgeCommand       = "/usr/bin/forge"
)

// creates all supporting resources required by build job
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

// creates build job service account when missing
func (r *ContainerImageBuildReconciler) checkServiceAccount(ctx context.Context, cib *forgev1alpha1.ContainerImageBuild) error {
	return r.withOwnedResource(ctx, cib, &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cib.Name,
			Namespace: cib.Namespace,
			Labels:    cib.Labels,
		},
	})
}

// creates build role when missing
func (r *ContainerImageBuildReconciler) checkRole(ctx context.Context, cib *forgev1alpha1.ContainerImageBuild) error {
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
			{
				APIGroups: []string{""},
				Resources: []string{"secrets"},
				Verbs:     []string{"get"},
			},
		},
	}

	if r.JobConfig.PodSecurityPolicy != "" {
		role.Rules = append(role.Rules, rbacv1.PolicyRule{
			APIGroups:     []string{"policy"},
			Resources:     []string{"podsecuritypolicies"},
			Verbs:         []string{"use"},
			ResourceNames: []string{r.JobConfig.PodSecurityPolicy},
		})
	}

	if r.JobConfig.SecurityContextConstraints != "" {
		role.Rules = append(role.Rules, rbacv1.PolicyRule{
			APIGroups:     []string{"security.openshift.io"},
			Resources:     []string{"securitycontextconstraints"},
			Verbs:         []string{"use"},
			ResourceNames: []string{r.JobConfig.SecurityContextConstraints},
		})
	}

	return r.withOwnedResource(ctx, cib, role)
}

// creates build job service role binding when missing
func (r *ContainerImageBuildReconciler) checkRoleBinding(ctx context.Context, cib *forgev1alpha1.ContainerImageBuild) error {
	return r.withOwnedResource(ctx, cib, &rbacv1.RoleBinding{
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
	})
}

// generates build job definition using container image build spec
func (r *ContainerImageBuildReconciler) createJobForBuild(ctx context.Context, cib *forgev1alpha1.ContainerImageBuild) error {
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
	for k, v := range cib.Annotations {
		podMeta.Annotations[k] = v
	}
	for k, v := range r.JobConfig.Annotations {
		podMeta.Annotations[k] = v
	}

	// setup security context
	podSecCtx := &corev1.PodSecurityContext{
		FSGroup: pointer.Int64Ptr(1000),
	}
	secCtx := &corev1.SecurityContext{
		RunAsUser: pointer.Int64Ptr(1000),
		SELinuxOptions: &corev1.SELinuxOptions{
			// TODO: this is currently required, because the default container SELinux rules
			// do not seem to allow the remount,ro system calls that containerd uses. "spc_t"
			// is a "special, super-privileged container" type (https://danwalsh.livejournal.com/74754.html)
			Type: "spc_t",
		},
	}
	if r.JobConfig.GrantFullPrivilege {
		podSecCtx.FSGroup = nil
		secCtx.RunAsUser = pointer.Int64Ptr(0)
		secCtx.Privileged = pointer.BoolPtr(true)
	}

	// setup volumes and mounts used by main container
	volumes := []corev1.Volume{
		{
			Name: "state-dir",
			VolumeSource: corev1.VolumeSource{
				EmptyDir: &corev1.EmptyDirVolumeSource{},
			},
		},
	}
	for _, volume := range r.JobConfig.Volumes {
		volumes = append(volumes, volume)
	}

	volumeMounts := []corev1.VolumeMount{
		{
			Name:      "state-dir",
			MountPath: config.GetStateDir(),
		},
	}
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
			TTLSecondsAfterFinished: pointer.Int32Ptr(0),
			Template: corev1.PodTemplateSpec{
				ObjectMeta: podMeta,
				Spec: corev1.PodSpec{
					ServiceAccountName: cib.Name,
					RestartPolicy:      corev1.RestartPolicyNever,
					InitContainers:     initContainers,
					SecurityContext:    podSecCtx,
					Containers: []corev1.Container{
						{
							Name:            "forge-build",
							Image:           r.JobConfig.Image,
							Command:         []string{rootlesskitCommand},
							Args:            r.prepareJobArgs(cib),
							Env:             r.JobConfig.EnvVar,
							SecurityContext: secCtx,
							VolumeMounts:    volumeMounts,
						},
					},
					Volumes: volumes,
				},
			},
		},
	}

	return r.withOwnedResource(ctx, cib, job)
}

// builds cli args required to launch forge in "build mode" inside a job
func (r *ContainerImageBuildReconciler) prepareJobArgs(cib *forgev1alpha1.ContainerImageBuild) []string {
	args := []string{
		forgeCommand,
		"build",
		fmt.Sprintf("--resource=%s", cib.Name),
		fmt.Sprintf("--enable-layer-caching=%t", r.JobConfig.EnableLayerCaching),
		fmt.Sprintf("--preparer-plugins-path=%s", r.JobConfig.PreparerPluginPath),
	}

	if r.JobConfig.BrokerOpts != nil {
		opts := r.JobConfig.BrokerOpts
		bs := []string{
			fmt.Sprintf("--message-broker=%s", opts.Broker),
			fmt.Sprintf("--amqp-queue=%s", opts.AmqpQueue),
			fmt.Sprintf("--amqp-uri=%s", opts.AmqpURI),
		}
		args = append(args, bs...)
	}

	return args
}

// checks if runtime object exists. if it does not exist, ownership is assigned to a container image build resource and
// the object is then created. otherwise, this procedure results in a no-op.
func (r *ContainerImageBuildReconciler) withOwnedResource(ctx context.Context, cib *forgev1alpha1.ContainerImageBuild, obj metav1.Object) error {
	err := r.Get(ctx, types.NamespacedName{Name: obj.GetName(), Namespace: obj.GetNamespace()}, obj.(runtime.Object).DeepCopyObject())
	if err == nil {
		return nil
	}
	if !apierrors.IsNotFound(err) {
		return err
	}

	if err := controllerutil.SetControllerReference(cib, obj, r.Scheme); err != nil {
		return err
	}
	return r.Create(ctx, obj.(runtime.Object))
}
