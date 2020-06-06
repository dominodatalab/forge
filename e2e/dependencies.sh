#!/usr/bin/env bash

set -e

wd="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null && pwd )"
# shellcheck source=utils.sh
source "$wd/utils.sh"

namespace="${NAMESPACE:-forge-test-$(tr -cd 'a-z0-9' < /dev/urandom | fold -w10 | head -n1)}"

info "Creating test namespace: $namespace"
kubectl get ns "$namespace" || kubectl create ns "$namespace"

info "Ensuring Helm repositories are configured"
helm repo add stable https://kubernetes-charts.storage.googleapis.com
helm repo add bitnami https://charts.bitnami.com/bitnami
helm repo add jetstack https://charts.jetstack.io
helm repo update

info "Installing cert-manager chart"
helm upgrade cert-manager jetstack/cert-manager \
  --install \
  --version v0.15.0 \
  --namespace "$namespace" \
  --set installCRDs=true \
  --wait

info "Generate custom root CA"
if ! kubectl get secret -n "$namespace" custom-root-ca &> /dev/null; then
  pushd e2e
  openssl genrsa -out rootCA.key 4096
  openssl req -x509 -new -nodes -key rootCA.key -sha256 -days 365 -out rootCA.crt \
    -subj "/C=US/ST=CA/L=San Francisco/O=Domino Data Lab, Inc./OU=Engineering/CN=dominodatalab.com"
  kubectl create secret tls custom-root-ca \
    --namespace "$namespace" \
    --key rootCA.key \
    --cert rootCA.crt
  popd
fi

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
helm upgrade docker-registry stable/docker-registry \
  --install \
  --version 1.9.2 \
  --namespace "$namespace" \
  --values e2e/helm_values/docker-registry-secure.yaml \
  --wait

info "Installing rabbitmq chart"
helm upgrade rabbitmq bitnami/rabbitmq \
  --install \
  --version 6.18.2 \
  --namespace "$namespace" \
  --set "persistence.enabled=false" \
  --set "rabbitmq.password=password" \
  --wait
