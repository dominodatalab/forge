package crd

import (
	"context"
	"encoding/json"

	"github.com/go-logr/logr"
	apixv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apixv1client "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/typed/apiextensions/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/yaml"

	"github.com/dominodatalab/forge"
	"github.com/dominodatalab/forge/internal/kubernetes"
)

const crdFilename = "config/crd/bases/forge.dominodatalab.com_containerimagebuilds.yaml"

var logger = zap.New()

func Apply(ctx context.Context) error {
	logger.Info("Initializing Kubernetes CRD client")

	crdClient, err := getCRDClient()
	if err != nil {
		return err
	}

	return createOrUpdateCRD(ctx, logger, crdClient)
}

func createOrUpdateCRD(ctx context.Context, logger logr.Logger, crdClient apixv1client.CustomResourceDefinitionInterface) error {
	crd, err := loadCRD(logger)
	if err != nil {
		return err
	}

	existing, err := crdClient.Get(ctx, crd.Name, metav1.GetOptions{})
	switch {
	case errors.IsNotFound(err):
		logger.Info("Existing CRD not found, creating", "name", crd.Name)
		_, err = crdClient.Create(ctx, crd, metav1.CreateOptions{})
	case err == nil: // update the crd
		// TODO: we currently do not check if the update is "safe" re: data loss as the documentation says we ought to.
		// https://kubernetes.io/docs/tasks/extend-kubernetes/custom-resources/custom-resource-definition-versioning/#upgrade-existing-objects-to-a-new-stored-version
		// https://github.com/operator-framework/operator-lifecycle-manager/blob/0.16.1/pkg/lib/crd/storage.go
		logger.Info("Existing CRD found, updating", "name", crd.Name)
		crd.SetResourceVersion(existing.ResourceVersion)
		_, err = crdClient.Update(ctx, crd, metav1.UpdateOptions{})
	default:
	}

	return err
}

func Delete(ctx context.Context) error {
	logger.Info("Initializing Kubernetes CRD client")

	crdClient, err := getCRDClient()
	if err != nil {
		return err
	}

	return deleteCRD(ctx, logger, crdClient)
}

func deleteCRD(ctx context.Context, logger logr.Logger, crdClient apixv1client.CustomResourceDefinitionInterface) error {
	crd, err := loadCRD(logger)
	if err != nil {
		return err
	}

	logger.Info("Deleting CRD", "name", crd.Name)
	return crdClient.Delete(ctx, crd.Name, metav1.DeleteOptions{})
}

func getCRDClient() (apixv1client.CustomResourceDefinitionInterface, error) {
	restCfg, err := kubernetes.LoadKubernetesConfig()
	if err != nil {
		return nil, err
	}

	client, err := apixv1client.NewForConfig(restCfg)
	if err != nil {
		return nil, err
	}
	crdClient := client.CustomResourceDefinitions()
	return crdClient, nil
}

func loadCRD(logger logr.Logger) (*apixv1.CustomResourceDefinition, error) {
	logger.Info("Loading existing CRD", "filename", crdFilename)

	yBytes, err := forge.CRDs.ReadFile(crdFilename)
	if err != nil {
		return nil, err
	}

	crdTmpl, err := yaml.YAMLToJSON(yBytes)
	if err != nil {
		return nil, err
	}

	crd := &apixv1.CustomResourceDefinition{}
	if err := json.Unmarshal(crdTmpl, crd); err != nil {
		return nil, err
	}
	return crd, nil
}
