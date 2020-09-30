package crd

import (
	"errors"
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

var logger = zapr.NewLogger(zap.NewNop())

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

	if err := createOrUpdateCRD(logger, fakeClient.ApiextensionsV1beta1().CustomResourceDefinitions()); err != nil {
		t.Error(err)
	}

	if !created {
		t.Errorf("New CRD was not created")
	}
}

func TestUpdateCRD(t *testing.T) {
	resourceVersion := "12345"
	fakeClient := fake.NewSimpleClientset(&apixv1beta1.CustomResourceDefinition{
		ObjectMeta: metav1.ObjectMeta{
			ResourceVersion: resourceVersion,
			Name:            "containerimagebuilds.forge.dominodatalab.com",
		},
	})

	updated := false
	fakeClient.PrependReactor("update", "*", func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
		updateAction := action.(k8stesting.UpdateAction)
		obj := updateAction.GetObject().(*apixv1beta1.CustomResourceDefinition)
		if obj.ResourceVersion != resourceVersion {
			t.Errorf("ResourceVersion was not passed through on update; received %v, expected %v", obj.ResourceVersion, resourceVersion)
		}

		updated = true
		return true, nil, nil
	})

	if err := createOrUpdateCRD(logger, fakeClient.ApiextensionsV1beta1().CustomResourceDefinitions()); err != nil {
		t.Error(err)
	}

	if !updated {
		t.Errorf("Existing CRD was not updated")
	}
}

func TestApplyCRDError(t *testing.T) {
	fakeClient := fake.NewSimpleClientset()

	expected := apierrors.NewInternalError(errors.New("thsi is an error"))
	fakeClient.PrependReactor("get", "*", func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
		return true, nil, expected
	})

	if err := createOrUpdateCRD(logger, fakeClient.ApiextensionsV1beta1().CustomResourceDefinitions()); err == nil || err != expected {
		t.Errorf("Received error %v did not match %v", err, expected)
	}
}

func TestDeleteCRD(t *testing.T) {
	fakeClient := fake.NewSimpleClientset(&apixv1beta1.CustomResourceDefinition{
		ObjectMeta: metav1.ObjectMeta{
			Name: "containerimagebuilds.forge.dominodatalab.com",
		},
	})

	deleted := false
	fakeClient.PrependReactor("delete", "*", func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
		deleted = true
		return true, nil, nil
	})

	if err := deleteCRD(logger, fakeClient.ApiextensionsV1beta1().CustomResourceDefinitions()); err != nil {
		t.Error(err)
	}

	if !deleted {
		t.Errorf("Existing CRD was not deleted")
	}
}
