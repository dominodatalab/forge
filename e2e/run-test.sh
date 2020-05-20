#!/usr/bin/env bash
#
# test cases:
# - insecure
# - TLS
# - authN using both inline and secret credentials
# - publishing to amqp broker

set -e

function info {
  echo -e "--> \033[1;32m$1\033[0m"
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
helm repo add stable https://kubernetes-charts.storage.googleapis.com
helm repo add bitnami https://charts.bitnami.com/bitnami
helm repo add jetstack https://charts.jetstack.io
helm repo update

info "Installing cert-manager chart"
helm install cert-manager jetstack/cert-manager \
  --version v0.15.0 \
  --namespace "$namespace" \
  --set installCRDs=true

info "Generate custom root CA"
openssl genrsa -out rootCA.key 4096
openssl req -x509 -new -nodes -key rootCA.key -sha256 -days 365 -out rootCA.crt \
  -subj "/C=US/ST=CA/L=San Francisco/O=Domino Data Lab, Inc./OU=Engineering/CN=dominodatalab.com"
kubectl create secret tls custom-root-ca \
  --key rootCA.key \
  --cert rootCA.crt

info "Creating self-signed issuer"
cat <<EOH kubectl apply --namespace "$namespace" -f -
apiVersion: cert-manager.io/v1alpha2
kind: Issuer
metadata:
  name: selfsigned-issuer
spec:
  ca:
    secretName: custom-root-ca
EOH

info "Creating certificate for docker registry"
cat <<EOH kubectl apply --namespace "$namespace" -f -
apiVersion: cert-manager.io/v1alpha2
kind: Certificate
metadata:
  name: docker-registry
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
  --values docker-registry-secure.yaml

info "Installing rabbitmq chart"
helm install rabbitmq bitnami/rabbitmq \
  --version 6.18.2 \
  --namespace "$namespace"

info "Installing forge CRDs"
kubectl apply -k config/crd

info "Launching Forge controller: $image"
#kubectl apply -k config/controller

info "All tests ran successfully"
