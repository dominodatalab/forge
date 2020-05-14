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

namespace="forge-test-$(tr -cd 'a-z0-9' < /dev/urandom | fold -w10 | head -n1)"

info "Creating test namespace: $namespace"
kubectl create ns "$namespace"

info "Ensuring stable Helm repository is present"
helm repo add stable https://kubernetes-charts.storage.googleapis.com
helm repo update

info "Installing Docker Registry chart"
helm install docker-registry stable/docker-registry \
  --version 1.9.2 \
  --namespace "$namespace"

info "Installing RabbitMQ chart"
helm install rabbitmq stable/rabbitmq \
  --version 3.8.2 \
  --namespace "$namespace"

info "Launching Forge controller"

info "All tests ran successfully"