package controllers

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/log"

	forgev1alpha1 "github.com/dominodatalab/forge/api/v1alpha1"
)

type testClient struct {
	client.Client
	listErr   bool
	deleteErr bool
}

func (c *testClient) List(ctx context.Context, list runtime.Object, opts ...client.ListOption) error {
	if c.listErr {
		return errors.New("something went wrong")
	}
	return c.Client.List(ctx, list, opts...)
}

func (c *testClient) Delete(ctx context.Context, obj runtime.Object, opts ...client.DeleteOption) error {
	if c.deleteErr {
		return errors.New("something went wrong")
	}
	return c.Client.Delete(ctx, obj, opts...)
}

func TestContainerImageBuildReconciler_RunGC(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, forgev1alpha1.AddToScheme(scheme))

	testController := func(listErr, deleteErr bool, testObjs ...runtime.Object) (*ContainerImageBuildReconciler, *record.FakeRecorder, func()) {
		fakeClient := fake.NewFakeClientWithScheme(scheme, testObjs...)
		fakeRecorder := record.NewFakeRecorder(10)
		cleanup := func() { close(fakeRecorder.Events) }

		controller := &ContainerImageBuildReconciler{
			Log: log.NullLogger{},
			Client: &testClient{
				Client:    fakeClient,
				listErr:   listErr,
				deleteErr: deleteErr,
			},
			Recorder: fakeRecorder,
		}

		return controller, fakeRecorder, cleanup
	}
	testObjs := func(addEligible bool) []runtime.Object {
		objs := []runtime.Object{
			&forgev1alpha1.ContainerImageBuild{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-cib-new",
				},
			},
			&forgev1alpha1.ContainerImageBuild{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-cib-initialized",
				},
				Status: forgev1alpha1.ContainerImageBuildStatus{
					State: forgev1alpha1.BuildStateInitialized,
				},
			},
			&forgev1alpha1.ContainerImageBuild{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-cib-building",
				},
				Status: forgev1alpha1.ContainerImageBuildStatus{
					State: forgev1alpha1.BuildStateBuilding,
				},
			},
		}

		if addEligible {
			objs = append(objs, []runtime.Object{
				&forgev1alpha1.ContainerImageBuild{
					ObjectMeta: metav1.ObjectMeta{
						Name:              "test-cib-failed",
						CreationTimestamp: metav1.Now(),
					},
					Status: forgev1alpha1.ContainerImageBuildStatus{
						State: forgev1alpha1.BuildStateFailed,
					},
				},
				&forgev1alpha1.ContainerImageBuild{
					ObjectMeta: metav1.ObjectMeta{
						Name:              "test-cib-failed-old",
						CreationTimestamp: metav1.NewTime(time.Now().Add(-1 * time.Hour)),
					},
					Status: forgev1alpha1.ContainerImageBuildStatus{
						State: forgev1alpha1.BuildStateFailed,
					},
				},
				&forgev1alpha1.ContainerImageBuild{
					ObjectMeta: metav1.ObjectMeta{
						Name:              "test-cib-completed",
						CreationTimestamp: metav1.Now(),
					},
					Status: forgev1alpha1.ContainerImageBuildStatus{
						State: forgev1alpha1.BuildStateCompleted,
					},
				},
				&forgev1alpha1.ContainerImageBuild{
					ObjectMeta: metav1.ObjectMeta{
						Name:              "test-cib-completed-old",
						CreationTimestamp: metav1.NewTime(time.Now().Add(-2 * time.Hour)),
					},
					Status: forgev1alpha1.ContainerImageBuildStatus{
						State: forgev1alpha1.BuildStateCompleted,
					},
				},
			}...)
		}

		return objs
	}
	testDefaultRetention := 2

	testCases := []struct {
		name      string
		retention int
		testObjs  []runtime.Object
		expected  []string
		listErr   bool
		deleteErr bool
		eventMsg  string
	}{
		{
			name:      "no_resources",
			retention: testDefaultRetention,
			testObjs:  []runtime.Object{},
			expected:  []string{},
		},
		{
			name:      "none_eligible",
			retention: testDefaultRetention,
			testObjs:  testObjs(false),
			expected:  []string{"test-cib-new", "test-cib-initialized", "test-cib-building"},
		},
		{
			name:      "oldest_eligible",
			retention: testDefaultRetention,
			testObjs:  testObjs(true),
			expected:  []string{"test-cib-new", "test-cib-initialized", "test-cib-building", "test-cib-completed", "test-cib-failed"},
		},
		{
			name:      "full_retention",
			testObjs:  testObjs(true),
			retention: 4,
			expected:  []string{"test-cib-new", "test-cib-initialized", "test-cib-building", "test-cib-completed", "test-cib-failed", "test-cib-completed-old", "test-cib-failed-old"},
		},
		{
			name:      "no_retention",
			retention: 0,
			testObjs:  testObjs(true),
			expected:  []string{"test-cib-new", "test-cib-initialized", "test-cib-building"},
		},
		{
			name:     "list_errors",
			listErr:  true,
			eventMsg: "Warning GarbageCollection Unable to list ContainerImageBuild resources",
		},
		{
			name:      "delete_errors",
			retention: 0,
			testObjs:  testObjs(true),
			deleteErr: true,
			eventMsg:  "Warning GarbageCollection Delete operation failed",
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			controller, fakeRecorder, cleanup := testController(tc.listErr, tc.deleteErr, tc.testObjs...)
			defer cleanup()

			controller.RunGC(tc.retention)

			if tc.listErr || tc.deleteErr {
				select {
				case actual := <-fakeRecorder.Events:
					assert.Equal(t, tc.eventMsg, actual)
				default:
					assert.Fail(t, "expected event during list error, got none")
				}
				return
			} else {
				select {
				case msg := <-fakeRecorder.Events:
					assert.Failf(t, "expected no event broadcast, got one", "event message: %q", msg)
				default: // happy path
				}
			}

			bl := &forgev1alpha1.ContainerImageBuildList{}
			require.NoError(t, controller.Client.List(context.TODO(), bl))

			var actual []string
			for _, item := range bl.Items {
				actual = append(actual, item.Name)
			}
			assert.ElementsMatch(t, tc.expected, actual)
		})
	}
}
