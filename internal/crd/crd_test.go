package crd

import (
	"testing"

	"github.com/go-logr/zapr"
	"go.uber.org/zap"
	apixv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	"k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/fake"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	k8stesting "k8s.io/client-go/testing"
)

func TestCreateCRD(t *testing.T) {
	fakeClient := fake.NewSimpleClientset()

	fakeClient.PrependReactor("get", "*", func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
		return true, nil, apierrors.NewNotFound(action.GetResource().GroupResource(), action.GetSubresource())
	})

	created := false
	fakeClient.PrependReactor("create", "*", func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
		created = true
		return true, nil, nil
	})

	if err := createOrUpdateCRD(zapr.NewLogger(zap.NewNop()), fakeClient.ApiextensionsV1beta1().CustomResourceDefinitions()); err != nil {
		t.Error(err)
	}

	if !created {
		t.Errorf("New CRD was not created")
	}
}

func TestUpdateCRD(t *testing.T) {
	fakeClient := fake.NewSimpleClientset()

	fakeClient.PrependReactor("get", "*", func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
		return true, &apixv1beta1.CustomResourceDefinition{
			ObjectMeta: metav1.ObjectMeta{
				Name: "containerimagebuild",
			},
		}, nil
	})

	updated := false
	fakeClient.PrependReactor("update", "*", func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
		updated = true
		return true, nil, nil
	})

	if err := createOrUpdateCRD(zapr.NewLogger(zap.NewNop()), fakeClient.ApiextensionsV1beta1().CustomResourceDefinitions()); err != nil {
		t.Error(err)
	}

	if !updated {
		t.Errorf("Existing CRD was not updated")
	}
}

func TestDeleteCRD(t *testing.T) {
	// no-op
}
