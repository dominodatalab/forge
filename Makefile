# Image URL to use all building/pushing image targets
IMG ?= quay.io/domino/forge:latest
# Produce CRDs that work back to Kubernetes 1.11 (no version conversion)
CRD_OPTIONS ?= "+crd"
# Add extra build flags
BUILD_FLAGS ?=

PWD=$(shell pwd)

TOOLS_FLAGS=-mod=mod -modfile=tools/go.mod

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

install_tools:
	go install $(TOOLS_FLAGS) sigs.k8s.io/controller-tools/cmd/controller-gen
	go install $(TOOLS_FLAGS) k8s.io/code-generator/cmd/client-gen
	go install $(TOOLS_FLAGS) github.com/golangci/golangci-lint/cmd/golangci-lint

# Generate manifests e.g. CRD, RBAC etc.
manifests:
	GOOS=linux GOFLAGS=-mod=mod controller-gen $(CRD_OPTIONS) paths="./..." output:crd:artifacts:config=config/crd/bases

# Run go fmt against code
fmt:
	go fmt ./...

# Run project linters
lint:
	golangci-lint run --timeout=5m

# Generate code
generate:
	client-gen -o ./tmp --output-package="github.com/dominodatalab/forge/internal" --clientset-name="clientset" --input-base="github.com/dominodatalab/forge/api" --input="forge/v1alpha1" --go-header-file="./hack/boilerplate.go.txt"
	cp -r ./tmp/github.com/dominodatalab/forge/* .
	controller-gen object:headerFile=./hack/boilerplate.go.txt paths="./api/..."
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
		--workdir /go/src/github.com/dominodatalab/forge golang:1.17-buster \
		sh -c "make install_tools manifests generate"

outdated:
	go list -mod=readonly -u -m -f '{{if and .Update (not .Indirect)}}{{.}}{{end}}' all
