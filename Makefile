# Image URL to use all building/pushing image targets
IMG ?= quay.io/domino/forge:latest
# Produce CRDs that work back to Kubernetes 1.11 (no version conversion)
CRD_OPTIONS ?= "crd:trivialVersions=true"
# Add extra build flags
BUILD_FLAGS ?=

# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

PWD=$(shell pwd)

all: manager

static:
	CGO_ENABLED=1 GOOS=linux GOARCH=amd64 GO111MODULE=on go build -o bin/forge -a -mod vendor $(BUILD_FLAGS)

# Run tests
test: generate fmt vet golangci-lint manifests
	go test ./... -coverprofile cover.out

# Build manager binary
manager: generate fmt vet
	go build -o bin/forge -mod vendor $(BUILD_FLAGS) main.go

# Run against the configured Kubernetes cluster in ~/.kube/config
run: generate fmt vet manifests
	go run ./main.go

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
manifests: controller-gen
	$(CONTROLLER_GEN) $(CRD_OPTIONS) paths="./..." output:crd:artifacts:config=config/crd/bases

# Run go fmt against code
fmt:
	go fmt ./...

# Run go vet against code
vet:
	go vet ./...

golangci-lint:
	golangci-lint run --skip-dirs docs

# Generate code
generate: controller-gen
	$(CONTROLLER_GEN) object:headerFile=./hack/boilerplate.go.txt paths="./..."

# Build the docker image
docker-build:
	docker build . -t ${IMG} $(ARGS)

# Push the docker image
docker-push:
	docker push ${IMG}

# Regenerate controller manifests and code using Docker (useful if on MacOS)
controller-regen-docker:
	docker run --rm -it -v ${PWD}:/forge \
		--workdir /forge golang:1.13-alpine3.12 \
		sh -c "apk add --no-cache build-base && make manifests generate"

# find or download controller-gen
# download controller-gen if necessary
controller-gen:
ifeq (, $(shell which controller-gen))
	@{ \
	set -e ;\
	CONTROLLER_GEN_TMP_DIR=$$(mktemp -d) ;\
	cd $$CONTROLLER_GEN_TMP_DIR ;\
	go mod init tmp ;\
	go get sigs.k8s.io/controller-tools/cmd/controller-gen@v0.2.4 ;\
	rm -rf $$CONTROLLER_GEN_TMP_DIR ;\
	}
CONTROLLER_GEN=$(GOBIN)/controller-gen
else
CONTROLLER_GEN=$(shell which controller-gen)
endif
