# Forge

[![CircleCI](https://circleci.com/gh/dominodatalab/forge.svg?style=shield)](https://app.circleci.com/pipelines/github/dominodatalab/forge)
[![Go Report Card](https://goreportcard.com/badge/github.com/dominodatalab/forge)](https://goreportcard.com/report/github.com/dominodatalab/forge)
[![PkgGoDev](https://pkg.go.dev/badge/mod/github.com/dominodatalab/forge)](https://pkg.go.dev/mod/github.com/dominodatalab/forge)

Forge is a Kubernetes controller designed to securely build OCI-compliant images
inside a cluster and push them to one or more target registries. This project
was derived from the work done in the [img][img] project and extended to support
build dispatch via a [custom resource definition][crd].

## Development

### MacOS

Because `forge` cannot run natively on MacOS due to the use of Linux-specific features in buildkit / runc, development is facilitated by the [skaffold](https://skaffold.dev/) project to develop inside Kubernetes.

#### Prerequisites

* [skaffold](https://skaffold.dev) - `brew install skaffold`
* [minikube](https://minikube.sigs.k8s.io/docs/) - `brew install minikube`  _note: kind is untested, but may also work_

  Ensure a `minikube` cluster is running and your Kubernetes context points to it.

* openssl - `brew install openssl`

  The default MacOS `openssl` is incompatible with the version used by cert-manager in development.
  Ensure `openssl` takes precendence in your PATH: `export PATH="/usr/local/opt/openssl@1.1/bin:$PATH"`.

* kustomize - `brew install kustomize`

#### Running the controller

To set up the necessary runtime dependencies for development (`docker-registry`, `rabbitmq`), run the following:

```
$ export NAMESPACE=forge-dev
$ e2e/dependencies.sh
```

Following completion, `skaffold` can be used to start up the dev-build-deploy cycle:

```
$ skaffold dev -p controller
```

_Note: by default skaffold will watch for changed files to rebuild and deploy the changes._

Test builds can be found in `e2e/builds`, for example:

```
$ kubectl create -n forge-dev -f e2e/builds/tls_with_basic_auth.yaml
```

#### Debugging the controller

Skaffold supports a built-in `debug` subcommand that uses [delve](https://github.com/go-delve/delve) to provide an interactive debugger.

To run, use:

```
$ skaffold debug --port-forward -p controller
```

#### Running a build job

As with the controller, set up the necessary runtime dependencies for development (`docker-registry`, `rabbitmq`):

```
$ export NAMESPACE=forge-dev
$ e2e/dependencies.sh
```

Following completion, `skaffold` can be used to start up the dev-build-deploy cycle:

```
$ kubectl apply -n forge-dev -f config/crd/bases
$ skaffold dev -p job --force=true
```

#### Debugging a build job

Skaffold supports a built-in `debug` subcommand that uses [delve](https://github.com/go-delve/delve) to provide an interactive debugger.

To run, use:

```
$ kubectl apply -n forge-dev -f config/crd/bases
$ skaffold debug --port-forward -p job --force=true
```

#### IntelliJ Debugging

In order to set breakpoints and pause the process from IntelliJ, configure a new "Go Remote" debug runtime with `localhost` and port `56268`.

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

### Usage

To add a new runtime plugin for Forge, place a file in `/usr/local/share/forge/plugins/` (by default) or specify it with `--preparer-plugins-path`.
<<<<<<< HEAD

[img]: https://github.com/genuinetools/img
[crd]: https://kubernetes.io/docs/tasks/extend-kubernetes/custom-resources/custom-resource-definitions/
=======
>>>>>>> f523a51... support skaffold as a development env on Mac OS
