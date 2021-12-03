#!/usr/bin/env bash

minikube start -p e2e --cpus=8
eval $(minikube -p e2e docker-env)
make docker-build
./e2e/run-test.sh quay.io/domino/forge:latest

# cleanup
rm ca-certificates.crt
echo "WARNING: changes made locally by kustomize. You may want to run 'git checkout' on affected files."
minikube delete -p e2e