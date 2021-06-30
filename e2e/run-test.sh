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

  info "Verifying that the $image_name image built by Forge has the expected files"
  local registry="localhost:32002"
  local registry_user=marge
  local registry_password=simpson

  info "Logging in to Docker"
  echo "$registry_password" | docker login $registry -u=$registry_user --password-stdin

  info "Copying files from image"
  local actual_dir=$(mktemp -d -t actual-XXXXXXXXXX)
  docker cp $(docker create $registry/$image_name):$src_dir "$actual_dir"

  info "Comparing files in $expected_dir with $actual_dir"
  if ! diff -r "$expected_dir" "$actual_dir"; then
    error  "diff failed"
    echo "EXPECTED:"
    ls -lahR "$expected_dir"
    echo "ACTUAL:"
    ls -lahR "$actual_dir"
    exit 1
  fi

  info "Test succeeded for image $image_name"
}

if [[ -z $1 ]]; then
  echo -e "Run integration tests against a Forge OCI image.\n\nUsage: $0 image"
  exit 1
fi

image="$1"
namespace="forge-test-$(head -c 1024 /dev/urandom | base64 | tr -cd "a-z0-9" | head -c 1)"

info "Creating test namespace: $namespace"
kubectl create ns "$namespace"

info "Ensuring Helm repositories are configured"
helm repo add stable https://charts.helm.sh/stable
helm repo add bitnami https://charts.bitnami.com/bitnami
helm repo add jetstack https://charts.jetstack.io
helm repo update

info "Installing cert-manager chart"
helm install cert-manager jetstack/cert-manager \
  --version v1.4.0 \
  --namespace "$namespace" \
  --set installCRDs=true \
  --wait

# install something to give the webhook time to start

info "Creating self-signed issuer"
set +e
retries=6
while (( retries > 0 )); do
  cat <<EOH | kubectl apply -f -
apiVersion: cert-manager.io/v1
kind: Issuer
metadata:
  name: selfsigned-issuer
  namespace: $namespace
spec:
  selfSigned: {}
EOH
  if [ $? -eq 0 ]; then
    break
  fi
  ((retries --))
  echo "retrying"
  sleep 10
done
if (( retries == 0 )); then
  echo "Failed to install selfsigned issuer"
  exit 1
fi

set -e

info "Creating certificate for docker registry"
cat <<EOH | kubectl apply -f -
apiVersion: cert-manager.io/v1
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


info "Launching Forge controller: $image"
pushd config/controller
kustomize edit set image quay.io/domino/forge="$image"
kustomize edit set namespace "$namespace"
# need kustomize v4.1.4 or higher -- https://github.com/kubernetes-sigs/kustomize/issues/4009
cat <<EOF >> kustomization.yaml
replacements:
- source:
    fieldPath: spec.template.spec.containers.[name=controller].image
    kind: Deployment
  targets:
  - fieldPaths:
    - spec.template.spec.containers.[name=controller].env.[name=BUILD_JOB_IMAGE].value
    select:
      kind: Deployment
+EOF
popd
kustomize build config/controller | kubectl apply -f -
kubectl wait deploy --for=condition=available \
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
verify_image variable-base-app /app $BASE_DIR/e2e/testdata/expected/simple-app

run_test "Build should run custom init container" \
          e2e/builds/init_container.yaml \
          test-init-container \
          "$namespace"
verify_image init-container-files /app $BASE_DIR/e2e/testdata/expected/init-container

info "All tests ran successfully"
