#!/usr/bin/env bash

set -e

wd="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null && pwd )"
# shellcheck source=utils.sh
source "$wd/utils.sh"
# shellcheck source=dependencies.sh
source "$wd/dependencies.sh"

function run_test {
  local test_name="$1"
  local yaml_file="$2"
  local resource_name="$3"

  info "Running test case: $test_name"
  kubectl apply -f "$yaml_file"

  local counter=0
  while true; do
    if [[ $counter -eq 5 ]]; then
      error "Test timeout reached"
      exit 1
    fi

    info "Waiting 10 secs for test to complete..."
    sleep 10

    local state
    state="$(kubectl get cib "$resource_name" -o jsonpath='{.status.state}')"
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

if [[ -z $1 ]]; then
  echo -e "Run integration tests against a Forge OCI image.\n\nUsage: $0 image"
  exit 1
fi

image="$1"

info "Installing forge CRDs"
kubectl apply -k config/crd

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
          test-tls-with-basic-auth
run_test "Build should pull base image from a private registry" \
          e2e/builds/private_base_image.yaml \
          test-private-base-image

info "All tests ran successfully"
