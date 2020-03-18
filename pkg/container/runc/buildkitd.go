package runc

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	appsv1 "k8s.io/api/apps/v1"
	apiv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

const (
	commonResourceName = "buildkitd"
	communicationPort  = 1234
)

func EnsureBuildkitDaemon() (hostURL string, err error) {
	clientset, err := getK8sClient()
	if err != nil {
		return "", err
	}
	ns, err := getNamespace()
	if err != nil {
		return "", err
	}

	labels := map[string]string{
		"app.kubernetes.io/name":       commonResourceName,
		"app.kubernetes.io/managed-by": "forge",
	}

	// ensure deployment
	deploymentsClient := clientset.AppsV1().Deployments(ns)
	if _, err := deploymentsClient.Get(commonResourceName, metav1.GetOptions{}); err != nil {
		if !errors.IsNotFound(err) {
			return "", err
		}

		probe := &apiv1.Probe{
			InitialDelaySeconds: 5,
			PeriodSeconds:       30,
			Handler: apiv1.Handler{
				Exec: &apiv1.ExecAction{
					Command: []string{
						"buildctl",
						"debug",
						"workers",
					},
				},
			},
		}
		deploy := &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:   commonResourceName,
				Labels: labels,
			},
			Spec: appsv1.DeploymentSpec{
				Replicas: int32Ptr(1),
				Selector: &metav1.LabelSelector{
					MatchLabels: labels,
				},
				Template: apiv1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{
						Labels: labels,
					},
					Spec: apiv1.PodSpec{
						Containers: []apiv1.Container{
							{
								Name:  commonResourceName,
								Image: "moby/buildkit:v0.6.4",
								Args: []string{
									"--addr", fmt.Sprintf("tcp://0.0.0.0:%d", communicationPort),
									"--addr", "unix:///run/buildkit/buildkitd.sock",
									"--debug",
								},
								Ports: []apiv1.ContainerPort{
									{
										Name:          "api",
										ContainerPort: communicationPort,
										Protocol:      apiv1.ProtocolTCP,
									},
								},
								ReadinessProbe: probe,
								LivenessProbe:  probe,
								SecurityContext: &apiv1.SecurityContext{
									Privileged: boolPtr(true),
								},
							},
						},
					},
				},
			},
		}

		if _, err := deploymentsClient.Create(deploy); err != nil {
			return "", err
		}
	}

	// ensure service
	servicesClient := clientset.CoreV1().Services(ns)
	svc, sErr := servicesClient.Get(commonResourceName, metav1.GetOptions{})
	if sErr != nil {
		if !errors.IsNotFound(sErr) {
			return "", sErr
		}

		service := &apiv1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name:   commonResourceName,
				Labels: labels,
			},
			Spec: apiv1.ServiceSpec{
				Ports: []apiv1.ServicePort{
					{
						Name:       "api",
						Protocol:   apiv1.ProtocolTCP,
						Port:       communicationPort,
						NodePort:   30138, // TODO remove me
						TargetPort: intstr.FromInt(communicationPort),
					},
				},
				Type:     apiv1.ServiceTypeNodePort, // TODO remove me
				Selector: labels,
			},
		}

		if svc, err = servicesClient.Create(service); err != nil {
			return "", err
		}
	}
	fmt.Printf("in-cluster addr: tcp://%s:%d", svc.Spec.ClusterIP, svc.Spec.Ports[0].Port)

	// NOTE replace w/ fmt.Sprintf("tcp://%s:%d", svc.Spec.ClusterIP, svc.Spec.Ports[0].Port) when running inside cluster
	return "tcp://192.168.64.78:30138", nil
}

func getK8sClient() (kubernetes.Interface, error) {
	var config *rest.Config

	config, err := clientcmd.BuildConfigFromFlags("", filepath.Join(os.Getenv("HOME"), ".kube", "config"))
	if err != nil {
		fmt.Println("cannot connect out-of-cluster, trying in-cluster connection")
		if config, err = rest.InClusterConfig(); err != nil {
			return nil, err
		}
	}

	return kubernetes.NewForConfig(config)
}

func getNamespace() (string, error) {
	bs, err := ioutil.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/namespace")
	if err != nil {
		if os.IsNotExist(err) {
			return apiv1.NamespaceDefault, nil
		}
		return "", err
	}

	return strings.TrimSpace(string(bs)), nil
}

func int32Ptr(i int32) *int32 {
	return &i
}

func boolPtr(b bool) *bool {
	return &b
}
