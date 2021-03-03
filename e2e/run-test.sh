#!/usr/bin/env bash

set -e

SCRIPT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
BASE_DIR=$(cd "${SCRIPT_DIR}/../" && pwd)

function info {
  echo -e "--> \033[1;32m$1\033[0m"
}

function error {
  echo -e "--> \033[1;31m$1\033[0m"
}

function run_test {
  local test_name="$1"
  local yaml_file="$2"
  local resource_name="$3"
  local namespace="$4"

  info "Running test case: $test_name"
  kubectl apply -f "$yaml_file" -n "$namespace"

  local counter=0
  while true; do
    if [[ $counter -eq 5 ]]; then
      error "Test timeout reached"
      exit 1
    fi

    info "Waiting 10 secs for test to complete..."
    sleep 10

    local state
    state="$(kubectl get cib "$resource_name" -n "$namespace" -o jsonpath='{.status.state}')"
    info "Current build state: '$state'"

    if [[ $state == "Completed" ]]; then
      info "Test succeeded"
      break
    fi

    if [[ $state == "Failed" ]]; then
      error "Test failed"
      kubectl logs --namespace "$namespace" --selector app.kubernetes.io/name=forge
      kubectl get cib "$resource_name" -o yaml
      exit 1
    fi

    counter=$((counter+1))
  done
}

function verify_image {
  local image_name=$1
  local src_dir=$2
  local expected_dir=$3

  info "Verifying that the image built by Forge has the expected files at the expected paths"
  local registry="localhost:32002"
  local registry_user=marge
  local registry_password=simpson

  info "Logging in to Docker"
  echo "$registry_password" | docker login $registry -u=$registry_user --password-stdin

  info "Copying files from image"
  local actual_dir=$(mktemp -d -t actual-XXXXXXXXXX)
  docker cp $(docker create $registry/$image_name):$src_dir "$actual_dir"

  info "Comparing files from the image with expected files"
  if ! diff -r "$expected_dir" "$actual_dir"; then
    error  "diff failed"
    echo "EXPECTED:"
    ls -lahR "$expected_dir"
    echo "ACTUAL:"
    ls -lahR "$actual_dir"
    exit 1
  fi

  info "Test succeeded"
}

if [[ -z $1 ]]; then
  echo -e "Run integration tests against a Forge OCI image.\n\nUsage: $0 image"
  exit 1
fi

image="$1"
namespace="forge-test-$(tr -cd 'a-z0-9' < /dev/urandom | fold -w10 | head -n1)"

info "Creating test namespace: $namespace"
kubectl create ns "$namespace"

info "Ensuring Helm repositories are configured"
helm repo add stable https://charts.helm.sh/stable
helm repo add bitnami https://charts.bitnami.com/bitnami
helm repo add jetstack https://charts.jetstack.io
helm repo update

info "Installing cert-manager chart"
helm install cert-manager jetstack/cert-manager \
  --version v0.15.0 \
  --namespace "$namespace" \
  --set installCRDs=true \
  --wait

info "Generate custom root CA"
pushd e2e
openssl genrsa -out rootCA.key 4096
openssl req -x509 -new -nodes -key rootCA.key -sha256 -days 365 -out rootCA.crt \
  -subj "/C=US/ST=CA/L=San Francisco/O=Domino Data Lab, Inc./OU=Engineering/CN=dominodatalab.com"
kubectl create secret tls custom-root-ca \
  --namespace "$namespace" \
  --key rootCA.key \
  --cert rootCA.crt
popd

info "Creating self-signed issuer"
cat <<EOH | kubectl apply -f -
apiVersion: cert-manager.io/v1alpha2
kind: Issuer
metadata:
  name: selfsigned-issuer
  namespace: $namespace
spec:
  ca:
    secretName: custom-root-ca
EOH

info "Creating certificate for docker registry"
cat <<EOH | kubectl apply -f -
apiVersion: cert-manager.io/v1alpha2
kind: Certificate
metadata:
  name: docker-registry
  namespace: $namespace
spec:
  secretName: docker-registry-tls
  dnsNames:
    - docker-registry
  issuerRef:
    name: selfsigned-issuer
EOH

info "Installing docker-registry chart"
helm install docker-registry stable/docker-registry \
  --version 1.9.2 \
  --namespace "$namespace" \
  --values e2e/helm_values/docker-registry-secure.yaml \
  --wait

info "Installing docker-registry-2 chart"
helm install docker-registry-2 stable/docker-registry \
  --version 1.9.2 \
  --namespace "$namespace" \
  --values e2e/helm_values/docker-registry-auth-only.yaml \
  --wait

info "Installing rabbitmq chart"
helm install rabbitmq bitnami/rabbitmq \
  --version 6.18.2 \
  --namespace "$namespace" \
  --wait

info "Launching Forge controller: $image"
pushd config/controller
kustomize edit set image quay.io/domino/forge="$image"
kustomize edit set namespace "$namespace"
popd
kubectl apply -k config/controller
kubectl wait pod --for=condition=ready \
  --namespace "$namespace" \
  --selector app.kubernetes.io/name=forge \
  --timeout 120s

run_test "Build should push to a private registry with TLS enabled" \
          e2e/builds/tls_with_basic_auth.yaml \
          test-tls-with-basic-auth \
          "$namespace"
run_test "Build should pull base image from a private registry" \
          e2e/builds/private_base_image.yaml \
          test-private-base-image \
          "$namespace"
verify_image variable-base-app /app $BASE_DIR/internal/archive/testdata/simple-app
run_test "Build should run custom init container" \
          e2e/builds/init_container.yaml \
          test-init-container \
          "$namespace"

info "All tests ran successfully"
