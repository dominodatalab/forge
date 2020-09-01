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
)

func TestContainerImageBuildReconciler_resourceLimits(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, forgev1alpha1.AddToScheme(scheme))
	require.NoError(t, batchv1.AddToScheme(scheme))

	fakeClient := fake.NewFakeClientWithScheme(scheme)
	fakeRecorder := record.NewFakeRecorder(10)

	controller := &ContainerImageBuildReconciler{
		Log:      log.NullLogger{},
		Client:   fakeClient,
		Recorder: fakeRecorder,
		JobConfig: &BuildJobConfig{
			Labels:      make(map[string]string),
			Annotations: make(map[string]string),
		},
		Scheme: scheme,
	}

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
					CpuQuota: 666,
					Memory:   "1G",
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
					CpuQuota: 0,
					Memory:   "",
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.cib.Name, func(t *testing.T) {
			if err := controller.createJobForBuild(context.TODO(), tc.cib); err != nil {
				t.Errorf("createJobForBuild() error = %v", err)
			}

			job := &batchv1.Job{}
			require.NoError(t, controller.Client.Get(context.TODO(), types.NamespacedName{Name: tc.cib.Name}, job))
			assert.Equal(t, job.Spec.Template.Spec.Containers[0].Resources, tc.resources)
		})
	}
}
