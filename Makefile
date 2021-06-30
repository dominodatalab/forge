# Image URL to use all building/pushing image targets
IMG ?= quay.io/domino/forge:latest
# Produce CRDs that work back to Kubernetes 1.11 (no version conversion)
CRD_OPTIONS ?= "crd:trivialVersions=true"
# Add extra build flags
BUILD_FLAGS ?=

PWD=$(shell pwd)

CONTROLLER_GEN=go run sigs.k8s.io/controller-tools/cmd/controller-gen
CLIENT_GEN=go run k8s.io/code-generator/cmd/client-gen
GOLANGCI_LINT=go run github.com/golangci/golangci-lint/cmd/golangci-lint

all: forge

static:
	CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build -o bin/forge -a -mod vendor $(BUILD_FLAGS) ./cmd/forge

precommit: generate fmt lint
	go mod tidy -v
	go mod vendor
	git update-index --refresh
	git diff-index --exit-code --name-status HEAD

# Run tests
test:
	go test -race ./... -coverprofile cover.out

# Build binary
forge:
	go build -o bin/forge -mod vendor $(BUILD_FLAGS) ./cmd/forge/

# Run against the configured Kubernetes cluster in ~/.kube/config
run:
	go run ./cmd/forge

# Install CRDs into a cluster
install: manifests
	kubectl apply -k config/crd

# Uninstall CRDs from a cluster
uninstall: manifests
	kubectl delete -k config/crd

# Deploy controller in the configured Kubernetes cluster in ~/.kube/config
deploy: manifests
	kubectl apply -k config/controller

# Generate manifests e.g. CRD, RBAC etc.
manifests:
	$(CONTROLLER_GEN) $(CRD_OPTIONS) paths="./..." output:crd:artifacts:config=config/crd/bases

# Run go fmt against code
fmt:
	go fmt ./...

# Run project linters
lint:
	$(GOLANGCI_LINT) run --timeout=5m

# Generate code
generate:
	$(CLIENT_GEN) -o ./tmp --output-package="github.com/dominodatalab/forge/internal" --clientset-name="clientset" --input-base="github.com/dominodatalab/forge/api" --input="forge/v1alpha1" --go-header-file="./hack/boilerplate.go.txt"
	cp -r ./tmp/github.com/dominodatalab/forge/* .
	$(CONTROLLER_GEN) object:headerFile=./hack/boilerplate.go.txt paths="./api/..."
	rm -rf ./tmp

# Build the docker image
docker-build:
	docker build . -t ${IMG} $(ARGS)

# Push the docker image
docker-push:
	docker push ${IMG}

# Regenerate controller manifests and code using Docker (useful if on MacOS)
controller-regen-docker:
	docker run --rm -it -v ${PWD}:/go/src/github.com/dominodatalab/forge \
		--workdir /go/src/github.com/dominodatalab/forge golang:1.16-buster \
		sh -c "make manifests generate"

outdated:
	go list -mod=readonly -u -m -f '{{if and .Update (not .Indirect)}}{{.}}{{end}}' all
