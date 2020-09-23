package crd

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/gobuffalo/packr/v2"
	apixv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	apixv1beta1client "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/typed/apiextensions/v1beta1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/yaml"
)

var crdTmpl []byte

const (
	crdDir      = "../../config/crd/bases"
	crdFilename = "forge.dominodatalab.com_containerimagebuilds.yaml"
)

func Apply() error {
	crdClient, err := getCRDClient()
	if err != nil {
		return err
	}

	crd := &apixv1beta1.CustomResourceDefinition{}
	if err := json.Unmarshal(crdTmpl, crd); err != nil {
		panic(err)
	}

	existing, err := crdClient.Get(crd.Name, metav1.GetOptions{})
	switch {
	case errors.IsNotFound(err): // create the crd
		if _, err := crdClient.Create(crd); err != nil {
			panic(err)
		}
	case err == nil: // update the crd
		crd.ResourceVersion = existing.ResourceVersion
		if _, err := crdClient.Update(crd); err != nil {
			panic(err)
		}
	default: // something went very wrong
		panic(err)
	}

	return nil
}

func Delete() error {
	crdClient, err := getCRDClient()
	if err != nil {
		return err
	}

	crd, err := loadCRD()
	if err != nil {
		return err
	}

	if err := crdClient.Delete(crd.Name, &metav1.DeleteOptions{}); err != nil {
		return err
	}

	return nil
}

func getCRDClient() (apixv1beta1client.CustomResourceDefinitionInterface, error) {
	h := os.Getenv("HOME")
	kubeconfig := filepath.Join(h, ".kube", "config")

	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		panic(err.Error())
	}

	client, err := apixv1beta1client.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}
	crdClient := client.CustomResourceDefinitions()
	return crdClient, nil
}

func loadCRD() (*apixv1beta1.CustomResourceDefinition, error) {
	crd := &apixv1beta1.CustomResourceDefinition{}
	if err := json.Unmarshal(crdTmpl, crd); err != nil {
		panic(err)
	}
	return crd, nil
}

func init() {
	box := packr.New("crd-box", crdDir)

	yBytes, err := box.Find(crdFilename)
	if err != nil {
		panic(err)
	}
	crdTmpl, err = yaml.YAMLToJSON(yBytes)
	if err != nil {
		panic(err)
	}
}
