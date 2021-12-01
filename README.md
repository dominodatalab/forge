# Forge

[![CircleCI](https://circleci.com/gh/dominodatalab/forge.svg?style=shield)](https://app.circleci.com/pipelines/github/dominodatalab/forge)
[![Go Report Card](https://goreportcard.com/badge/github.com/dominodatalab/forge)](https://goreportcard.com/report/github.com/dominodatalab/forge)
[![PkgGoDev](https://pkg.go.dev/badge/mod/github.com/dominodatalab/forge)](https://pkg.go.dev/mod/github.com/dominodatalab/forge)

Forge is a Kubernetes controller designed to securely build OCI-compliant images
inside a cluster and push them to one or more target registries. This project
was derived from the work done in the [img][img] project and extended to support
build dispatch via a [custom resource definition][crd].

## Development

### Requirements

- Linux
- go 1.16
- golangci-lint
- kubebuilder
- A Kubernetes cluster with kubectl access configured (minikube suggested)

### Local setup

Start Kubernetes cluster:

```
minikube start --insecure-registry="localhost:32002"
```

Install helm on the host:

```
export HELM_VERSION="v3.6.1"
curl -L https://get.helm.sh/helm-$HELM_VERSION-linux-amd64.tar.gz | tar -xvz -C /tmp
sudo mv /tmp/linux-amd64/helm /usr/local/bin/
helm repo add stable https://charts.helm.sh/stable
helm repo update
```

Install Docker registry:

```
helm upgrade -i docker-registry stable/docker-registry \
  --values e2e/helm_values/docker-registry-auth-only.yaml \
  --wait
```

Create Docker registry auth secret:

```
kubectl apply -f config/samples/docker-registry-auth.yaml
```

Install local tools

```
make install_tools
```

Prepare cluster for running Forge builds by installing the CRD:

```
make install
```

(Repeat this step if you modify the custom resource definition.)

### Local edit / compile / test workflow

Run tests:

```
make test
```

Connect the host's Docker client to minikube's Docker server so that the image that will be built will be available in minikube:

```
eval $(minikube docker-env)
```

Build Forge image. This will default to building the image as `quay.io/domino/forge:latest`.

```
make docker-build
```

In a different terminal, run Forge's controller on the host, specifying the image from the previous step.
Minikube will not pull this image because it will already be present in the cluster. If you're only changing
the build job, then you can keep the controller running - the build job image will be replaced in place by
the commands above.

```
go run ./cmd/forge --build-job-image quay.io/domino/forge:latest
```

If you need to change the image name for any reason, use the `IMG` environment variable.
If you use two terminals, make sure  you use the same value in both.

```
export IMG=test-forge:$(date +%s)
make docker-build
go run ./cmd/forge --build-job-image $IMG
```

On a different terminal, create a CIB (container image build) resource to trigger a build:

```
kubectl apply -f config/samples/forge_v1alpha1_containerimagebuild.yaml
```

Watch for build pods:

```
kubectl get pods | grep build
```

Follow the build logs:

```
kubectl logs -f build-init-container-sample-jmnjf
```

Confirm that the image was built and uploaded to the Docker registry:

```
curl -u marge:simpson $(minikube ip):32002/v2/_catalog
```

To repeat the build with the same CIB resource name, first delete the old resource so that the Forge controller will destroy
the job and pod, and then create it again:

```
kubectl delete -f config/samples/forge_v1alpha1_containerimagebuild.yaml
# OR
kubectl delete cib example-build

# Reapply
kubectl apply -f config/samples/forge_v1alpha1_containerimagebuild.yaml
```

### Inspecting generated images

Since minikube was started with an option to allow a local insecure Docker registry on port 32002, a minikube shell
session will be able to work with images that were pushed to that registry:

```
minikube ssh
echo 'simpson' | docker login localhost:32002 -u marge --password-stdin
docker pull localhost:32002/init-container-sample
```

## Preparer Plugins

Forge supports the inclusion of custom plugins for the "preparation" phase of a build (between the initialization of the context and the actual image build).
This functionality is built with the [go-plugin](https://github.com/hashicorp/go-plugin) framework from Hashicorp.

### Creation

[example/preparer_plugin.go](./docs/example/preparer_plugin.go) has the necessary structure for creating a new preparer plugin.
Functionality is implemented through two primary functions:

`Prepare(contextPath string, pluginData map[string]string) error`

Prepare runs between the context creation and image build starting. `contextPath` is an absolute path to the context for the build.
`pluginData` is the key-value data passed through the [ContainerImageBuild](./config/crd/bases/forge.dominodatalab.com_containerimagebuilds.yaml#L77-L82).

`Cleanup() error`

Cleanup runs after the build has finished (successfully or otherwise).

### Using

To add a new runtime plugin for Forge, place a file in `/usr/local/share/forge/plugins/` (by default) or specify it with `--preparer-plugins-path`.

[img]: https://github.com/genuinetools/img
[crd]: https://kubernetes.io/docs/tasks/extend-kubernetes/custom-resources/custom-resource-definitions/
