#!/usr/bin/env bash

minikube start -p e2e --cpus=8
eval $(minikube -p e2e docker-env)
make docker-build
./e2e/run-test.sh quay.io/domino/forge:latest
minikube delete -p e2e