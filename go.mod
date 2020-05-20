module github.com/dominodatalab/forge

go 1.14

replace github.com/containerd/containerd v1.3.0-0.20190212172151-f5b0fa220df8 => github.com/containerd/containerd v1.2.1-0.20190212172151-f5b0fa220df8

replace github.com/hashicorp/go-immutable-radix => github.com/tonistiigi/go-immutable-radix v0.0.0-20170803185627-826af9ccf0fe

require (
	github.com/containerd/console v0.0.0-20180822173158-c12b1e7919c1
	github.com/containerd/containerd v1.3.0-0.20190212172151-f5b0fa220df8
	github.com/docker/cli v0.0.0-20190913211141-95327f4e6241
	github.com/docker/distribution v2.7.1+incompatible
	github.com/docker/docker v1.4.2-0.20190916154449-92cc603036dd
	github.com/docker/go-metrics v0.0.1 // indirect
	github.com/genuinetools/reg v0.16.1
	github.com/go-logr/logr v0.1.0
	github.com/gogo/googleapis v1.1.0 // indirect
	github.com/gorilla/mux v1.7.4 // indirect
	github.com/h2non/filetype v1.0.12
	github.com/mitchellh/hashstructure v1.0.0 // indirect
	github.com/moby/buildkit v0.4.0
	github.com/onsi/ginkgo v1.8.0
	github.com/onsi/gomega v1.5.0
	github.com/opencontainers/image-spec v1.0.1
	github.com/opencontainers/runc v1.0.0-rc2.0.20181113215238-10d38b660a77
	github.com/opencontainers/runtime-spec v1.0.1 // indirect
	github.com/opentracing-contrib/go-stdlib v0.0.0-20180702182724-07a764486eb1 // indirect
	github.com/opentracing/opentracing-go v1.0.2 // indirect
	github.com/pkg/errors v0.8.1
	github.com/sirupsen/logrus v1.4.2
	github.com/spf13/cobra v0.0.5
	github.com/streadway/amqp v0.0.0-20200108173154-1c71cc93ed71
	github.com/stretchr/testify v1.3.0
	github.com/syndtr/gocapability v0.0.0-20180916011248-d98352740cb2 // indirect
	go.etcd.io/bbolt v1.3.1-etcd.8
	golang.org/x/sync v0.0.0-20190911185100-cd5d95a43a6e
	google.golang.org/grpc v1.23.1
	k8s.io/api v0.0.0-20190918155943-95b840bb6a1f
	k8s.io/apimachinery v0.0.0-20190913080033-27d36303b655
	k8s.io/client-go v0.0.0-20190918160344-1fbdaa4c8d90
	sigs.k8s.io/controller-runtime v0.4.0
)
