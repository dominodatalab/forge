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

  cat "$yaml_file" | \
    TEST_NAMESPACE=$namespace TEST_RESOURCE_NAME=$resource_name \
    envsubst '${TEST_NAMESPACE} ${TEST_RESOURCE_NAME}' | \
    kubectl apply -n "$namespace" -f -

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

function registry_login {
  local registry=$1
  local registry_user=marge
  local registry_password=simpson

  info "Logging in to Docker"
  echo "$registry_password" | docker login $registry -u=$registry_user --password-stdin
}

function verify_image {
  local image_name=$1
  local src_dir=$2
  local expected_dir=$3

  info "Verifying that the $image_name image built by Forge has the expected files"

  info "Copying files from image"
  local actual_dir=$(mktemp -d -t actual-XXXXXXXXXX)
  docker cp $(docker create $image_name):$src_dir "$actual_dir"

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

set -e -o pipefail

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

info "Creating generated CA bundle"
cp /etc/ssl/certs/ca-certificates.crt ca-certificates.crt
kubectl -n $namespace get secrets docker-registry-tls -o=jsonpath='{.data.ca\.crt}' | base64 -d >> ca-certificates.crt
kubectl -n $namespace create cm domino-generated-ca --from-file=ca-certificates.crt

info "Launching Forge controller: $image"
pushd config/controller
kustomize edit set image quay.io/domino/forge="$image"
kustomize edit set namespace "$namespace"
popd
kubectl -n "$namespace" create secret generic forge --from-literal=AZURE_TENANT_ID=$AZURE_TENANT_ID --from-literal=AZURE_CLIENT_ID=$AZURE_CLIENT_ID --from-literal=AZURE_CLIENT_SECRET=$AZURE_CLIENT_SECRET
kustomize build config/controller | kubectl apply -f -
kubectl wait deploy --for=condition=available \
  --namespace "$namespace" \
  --selector app.kubernetes.io/name=forge \
  --timeout 120s

registry="localhost:32002"
registry_login $registry

run_test "Build should push to a private registry with TLS enabled" \
          e2e/builds/tls_with_basic_auth.yaml \
          test-tls-with-basic-auth \
          "$namespace"
verify_image "$registry/simple-app" /app $BASE_DIR/e2e/testdata/expected/simple-app

run_test "Build should pull base image from a private registry" \
          e2e/builds/private_base_image.yaml \
          test-private-base-image \
          "$namespace"
verify_image "$registry/variable-base-app" /app $BASE_DIR/e2e/testdata/expected/simple-app

run_test "Build should run custom init container" \
          e2e/builds/init_container.yaml \
          test-init-container \
          "$namespace"
verify_image "$registry/init-container-files" /app $BASE_DIR/e2e/testdata/expected/init-container

run_test "Build pull base from registry in secret but not explicitly configured" \
          e2e/builds/all_registries_from_secret.yaml \
          test-all-registries-from-secret \
          "$namespace"
verify_image "$registry/all-registries-from-secret-app" /app $BASE_DIR/e2e/testdata/expected/simple-app

if [ -n "$ACR_REGISTRY" ]; then

 sed -i -e "s/ACR/$ACR_REGISTRY/g" e2e/builds/acr/*.yaml

 run_test "Build should push to ACR" \
   e2e/builds/acr/push.yaml \
   test-push-acr \
   "$namespace"

 run_test "Build should pull from ACR" \
   e2e/builds/acr/pull.yaml \
   test-pull-acr \
   "$namespace"

 az acr login -n ${ACR_REGISTRY%%.*}
 verify_image "$ACR_REGISTRY/variable-base-app" /app $BASE_DIR/e2e/testdata/expected/simple-app
fi

info "All tests ran successfully"
