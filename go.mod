module github.com/dominodatalab/forge

go 1.14

replace github.com/containerd/containerd v1.3.0-0.20190212172151-f5b0fa220df8 => github.com/containerd/containerd v1.2.1-0.20190212172151-f5b0fa220df8

replace github.com/hashicorp/go-immutable-radix => github.com/tonistiigi/go-immutable-radix v0.0.0-20170803185627-826af9ccf0fe

require (
	github.com/containerd/console v0.0.0-20180822173158-c12b1e7919c1
	github.com/containerd/containerd v1.3.0-0.20190212172151-f5b0fa220df8
	github.com/genuinetools/img v0.5.7
	github.com/go-logr/logr v0.1.0
	github.com/h2non/filetype v1.0.12
	github.com/moby/buildkit v0.4.0
	github.com/onsi/ginkgo v1.8.0
	github.com/onsi/gomega v1.5.0
	github.com/opencontainers/runc v1.0.0-rc2.0.20181113215238-10d38b660a77
	github.com/sirupsen/logrus v1.4.2
	github.com/stretchr/testify v1.3.0
	golang.org/x/sync v0.0.0-20190911185100-cd5d95a43a6e
	k8s.io/apimachinery v0.0.0-20190913080033-27d36303b655
	k8s.io/client-go v0.0.0-20190918160344-1fbdaa4c8d90
	sigs.k8s.io/controller-runtime v0.4.0
)
