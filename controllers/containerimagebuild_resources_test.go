package controllers

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/log"

	forgev1alpha1 "github.com/dominodatalab/forge/api/v1alpha1"
	"github.com/dominodatalab/forge/internal/message"
)

func TestContainerImageBuildReconciler_resourceLimits(t *testing.T) {
	controller := makeController(t)

	testCases := []struct {
		cib       *forgev1alpha1.ContainerImageBuild
		resources corev1.ResourceRequirements
	}{
		{
			cib: &forgev1alpha1.ContainerImageBuild{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-cib-resource-quota",
				},
				Spec: forgev1alpha1.ContainerImageBuildSpec{
					CPU:    "666m",
					Memory: "1G",
				},
			},
			resources: corev1.ResourceRequirements{
				Limits: corev1.ResourceList{
					"memory": resource.MustParse("1G"),
					"cpu":    resource.MustParse("666m"),
				},
				Requests: corev1.ResourceList{
					"memory": resource.MustParse("1G"),
					"cpu":    resource.MustParse("666m"),
				},
			},
		},
		{
			cib: &forgev1alpha1.ContainerImageBuild{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-cib-no-resource-quota",
				},
				Spec: forgev1alpha1.ContainerImageBuildSpec{
					CPU:    "",
					Memory: "",
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.cib.Name, func(t *testing.T) {
			require.NoError(t, controller.createJobForBuild(context.TODO(), tc.cib))

			job := &batchv1.Job{}
			require.NoError(t, controller.Client.Get(context.TODO(), types.NamespacedName{Name: tc.cib.Name}, job))
			assert.Equal(t, job.Spec.Template.Spec.Containers[0].Resources, tc.resources)
		})
	}
}

func TestContainerImageBuildReconciler_buildContextVolume(t *testing.T) {
	controller := makeController(t)

	cib := &forgev1alpha1.ContainerImageBuild{}
	require.NoError(t, controller.createJobForBuild(context.Background(), cib))

	job := &batchv1.Job{}
	require.NoError(t, controller.Client.Get(context.Background(), types.NamespacedName{Name: cib.Name}, job))

	expected := corev1.Volume{
		Name: "build-context-dir",
		VolumeSource: corev1.VolumeSource{
			EmptyDir: &corev1.EmptyDirVolumeSource{},
		},
	}

	assert.Contains(t, job.Spec.Template.Spec.Volumes, expected)
}

func TestContainerImageBuildReconciler_buildContextVolumeMount(t *testing.T) {
	controller := makeController(t)

	cib := &forgev1alpha1.ContainerImageBuild{}
	require.NoError(t, controller.createJobForBuild(context.Background(), cib))

	job := &batchv1.Job{}
	require.NoError(t, controller.Client.Get(context.Background(), types.NamespacedName{Name: cib.Name}, job))

	expected := corev1.VolumeMount{
		Name:      "build-context-dir",
		ReadOnly:  false,
		MountPath: "/mnt/build",
	}

	assert.Contains(t, job.Spec.Template.Spec.Containers[0].VolumeMounts, expected)
}

func TestContainerImageBuildReconciler_prepareJobArgs(t *testing.T) {
	tests := []struct {
		name      string
		jobConfig *BuildJobConfig
		want      string
	}{
		{
			name:      "rootless",
			jobConfig: &BuildJobConfig{},
			want:      "rootlesskit /usr/bin/forge build --resource=test-cib --enable-layer-caching=false",
		},
		{
			name:      "privileged",
			jobConfig: &BuildJobConfig{GrantFullPrivilege: true},
			want:      "/usr/bin/forge build --resource=test-cib --enable-layer-caching=false",
		},
		{
			name:      "istio",
			jobConfig: &BuildJobConfig{EnableIstioSupport: true},
			want:      "rootlesskit /usr/bin/forge build --resource=test-cib --enable-layer-caching=false \nEXIT_CODE=$?; wget -qO- --post-data \"\" http://localhost:15020/quitquitquit; exit $EXIT_CODE",
		},
		{
			name: "broker opts",
			jobConfig: &BuildJobConfig{BrokerOpts: &message.Options{
				Broker:    "my-broker",
				AmqpURI:   "amqp://uri:5672",
				AmqpQueue: "my-queue",
			}},
			want: "rootlesskit /usr/bin/forge build --resource=test-cib --enable-layer-caching=false --message-broker=my-broker --amqp-uri=amqp://uri:5672 --amqp-queue=my-queue",
		},
		{
			name:      "preparer plugins path",
			jobConfig: &BuildJobConfig{PreparerPluginPath: "/path/to/plugins"},
			want:      "rootlesskit /usr/bin/forge build --resource=test-cib --enable-layer-caching=false --preparer-plugins-path=/path/to/plugins",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &ContainerImageBuildReconciler{
				JobConfig: tt.jobConfig,
			}
			got := r.prepareJobArgs(&forgev1alpha1.ContainerImageBuild{
				ObjectMeta: metav1.ObjectMeta{Name: "test-cib"},
			})

			assert.Equal(t, []string{"-c", tt.want}, got)
		})
	}
}

func makeController(t *testing.T) ContainerImageBuildReconciler {
	scheme := runtime.NewScheme()
	require.NoError(t, forgev1alpha1.AddToScheme(scheme))
	require.NoError(t, batchv1.AddToScheme(scheme))

	fakeClient := fake.NewFakeClientWithScheme(scheme)
	fakeRecorder := record.NewFakeRecorder(10)

	return ContainerImageBuildReconciler{
		Log:      log.NullLogger{},
		Client:   fakeClient,
		Recorder: fakeRecorder,
		JobConfig: &BuildJobConfig{
			Labels:      make(map[string]string),
			Annotations: make(map[string]string),
		},
		Scheme: scheme,
	}
}
