#!/usr/bin/env bash

set -e

SCRIPT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
BASE_DIR=$(cd "${SCRIPT_DIR}/../" && pwd)

# shellcheck source=utils.sh
source "$SCRIPT_DIR/utils.sh"
# shellcheck source=dependencies.sh
source "$SCRIPT_DIR/dependencies.sh"

function run_test {
  local test_name="$1"
  local yaml_file="$2"
  local expected_state="${3:-Completed}"

  info "Running test case: $test_name"
  cib_name=$(kubectl create --namespace "$namespace" -f "$yaml_file" -o name)
  info "Created image build: $cib_name"

  local counter=0
  while true; do
    if [[ $counter -eq 5 ]]; then
      error "Test timeout reached"
      kubectl logs --namespace "$namespace" --selector app.kubernetes.io/name=forge
      kubectl get --namespace "$namespace" "$cib_name" -o yaml
      exit 1
    fi

    info "Waiting 10 secs for test to complete..."
    sleep 10

    local state
    state="$(kubectl get --namespace "$namespace" "$cib_name" -o jsonpath='{.status.state}')"
    completed_at="$(kubectl get --namespace "$namespace" "$cib_name" -o jsonpath='{.status.buildCompletedAt}')"
    info "Current build state: '$state'"

    if [[ $state == "$expected_state" ]]; then
      info "Test succeeded"
      break
    fi

    if [[ -n "$completed_at" ]]; then
      error "Test failed"
      kubectl logs --namespace "$namespace" --selector app.kubernetes.io/name=forge
      kubectl get --namespace "$namespace" "$cib_name" -o yaml
      exit 1
    fi

    counter=$((counter+1))
  done
}

function verify_image {
  info "Verifying that the image built by Forge has the expected files at the expected paths"
  local test_dir="/tmp/$namespace/verify_image"

  info "Creating a service exposing the Docker registry with no TLS on node port 32002"
  kubectl apply -f "$SCRIPT_DIR/docker-registry-2-nodeport-service.yaml" --namespace "$namespace"

  info "Logging in to Docker"
  echo "simpson" | docker login localhost:32002 -u=marge --password-stdin

  info "Copying files from image"
  mkdir -p "$test_dir/actual"
  docker cp $(docker create localhost:32002/variable-base-app:latest):/app/app.py "$test_dir/actual/app.py"

  info "Extracting files that are expected to be in the image"
  mkdir -p "$test_dir/expected"
  tar -xf "$BASE_DIR/internal/archive/testdata/simple-app.tar" -C "$test_dir/expected"

  info "Comparing files from the image with expected files"
  if ! diff "$test_dir/expected/app.py" "$test_dir/actual/app.py"; then
    error  "diff failed"
    ls -lahR "$test_dir"
    exit 1
  fi

  info "Test succeeded"
}

if [[ -z $1 ]]; then
  echo -e "Run integration tests against a Forge OCI image.\n\nUsage: $0 image"
  exit 1
fi

image="$1"
info "Launching Forge controller: $image"
pushd config/controller/controller
kustomize edit set image quay.io/domino/forge="$image"
kustomize edit set namespace "$namespace"
popd

kubectl apply -k config/controller/controller
kubectl wait deploy --for=condition=available \
  --namespace "$namespace" \
  --selector app.kubernetes.io/name=forge \
  --timeout 120s

run_test "Build should push to a private registry with TLS enabled" \
          e2e/builds/tls_with_basic_auth.yaml
run_test "Build should pull base image from a private registry" \
          e2e/builds/private_base_image.yaml
verify_image
run_test "Fail to build image with a size limit" \
          e2e/builds/image_size_limit.yaml \
          "Failed"

info "All tests ran successfully"
