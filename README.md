# Forge

Forge is project that provides a means to build container images inside
Kubernetes according to the OCI image format specification and push them to a
designated distribution registry.```

## Local Development

1. Launch minikube – `minikube start`
1. Install buildkit – `kubectl apply -f test/manifests/buildkitd-pod-service.yaml`
1. Install Docker registry – `kubectl apply -f test/manifests/registry.yaml`
1. Launch forge – `make run`