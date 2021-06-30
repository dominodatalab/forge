package crd

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/go-logr/zapr"
	"go.uber.org/zap"
	apixv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/fake"
	apixv1client "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/typed/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	k8stesting "k8s.io/client-go/testing"
)

func init() {
	logger = zapr.NewLogger(zap.NewNop())
}

func TestCreateCRD(t *testing.T) {
	fakeClient := fake.NewSimpleClientset()

	crdClient := fakeClient.ApiextensionsV1().CustomResourceDefinitions()
	if err := createOrUpdateCRD(context.Background(), logger, crdClient); err != nil {
		t.Error(err)
	}

	crds, err := collectCRDs(crdClient)
	if err != nil {
		t.Fatalf("collecting crds: %v", err)
	}

	if e, a := []string{"containerimagebuilds.forge.dominodatalab.com"}, crds; !reflect.DeepEqual(e, a) {
		t.Errorf("missing CRDs want:%v got:%v", e, a)
	}
}

func TestUpdateCRD(t *testing.T) {
	initialCRD := &apixv1.CustomResourceDefinition{
		ObjectMeta: metav1.ObjectMeta{
			ResourceVersion: "12345",
			Name:            "containerimagebuilds.forge.dominodatalab.com",
		},
	}
	fakeClient := fake.NewSimpleClientset(initialCRD)

	crdClient := fakeClient.ApiextensionsV1().CustomResourceDefinitions()
	if err := createOrUpdateCRD(context.Background(), logger, crdClient); err != nil {
		t.Error(err)
	}

	ctx := context.Background()
	crd, err := crdClient.Get(ctx, "containerimagebuilds.forge.dominodatalab.com", metav1.GetOptions{})
	if err != nil {
		t.Fatalf("get failed %v", err)
	}

	if crd.ResourceVersion != "12345" {
		t.Errorf("wrong resource version: %s", crd.ResourceVersion)
	}
	if reflect.DeepEqual(initialCRD.TypeMeta, crd.TypeMeta) {
		t.Errorf("not updated: %v", crd)
	}
}

func TestApplyCRDError(t *testing.T) {
	fakeClient := fake.NewSimpleClientset()

	expected := errors.New("an error")
	fakeClient.PrependReactor("get", "customresourcedefinitions", func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
		return true, nil, expected
	})

	if err := createOrUpdateCRD(context.Background(), logger, fakeClient.ApiextensionsV1().CustomResourceDefinitions()); err != expected {
		t.Errorf("Wrong error: got: %v, want %v", err, expected)
	}
}

func TestDeleteCRD(t *testing.T) {
	fakeClient := fake.NewSimpleClientset()
	crdClient := fakeClient.ApiextensionsV1().CustomResourceDefinitions()
	ctx := context.Background()
	if err := createOrUpdateCRD(ctx, logger, crdClient); err != nil {
		t.Error(err)
	}

	createdCRDs, err := collectCRDs(crdClient)
	if err != nil {
		t.Fatalf("created crds faiiled %v", err)
	}
	if len(createdCRDs) != 1 {
		t.Fatalf("created failed: %v", createdCRDs)
	}

	if err := deleteCRD(ctx, logger, crdClient); err != nil {
		t.Error(err)
	}

	deletedCRDs, err := collectCRDs(crdClient)
	if err != nil {
		t.Fatalf("deleted crds failed %v", err)
	}

	if len(deletedCRDs) != 0 {
		t.Fatalf("delete failed: %v", deletedCRDs)
	}
}

func TestDeleteCRDError(t *testing.T) {
	fakeClient := fake.NewSimpleClientset()
	expected := errors.New("an error")
	fakeClient.PrependReactor("delete", "customresourcedefinitions", func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
		return true, nil, expected
	})

	crdClient := fakeClient.ApiextensionsV1().CustomResourceDefinitions()
	ctx := context.Background()
	if err := createOrUpdateCRD(ctx, logger, crdClient); err != nil {
		t.Error(err)
	}
	if err := deleteCRD(ctx, logger, crdClient); err != expected {
		t.Errorf("Received error: want %v, got %v", expected, err)
	}
}

func collectCRDs(crdClient apixv1client.CustomResourceDefinitionInterface) ([]string, error) {
	ctx := context.Background()
	crds, err := crdClient.List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	forgeCrds := []string{}
	for _, crd := range crds.Items {
		if crd.Spec.Group == "forge.dominodatalab.com" {
			forgeCrds = append(forgeCrds, crd.Name)
		}
	}
	return forgeCrds, nil
}
