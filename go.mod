module github.com/dominodatalab/forge

go 1.16

// Overrides should match https://github.com/moby/buildkit/blob/<release>/go.mod
replace (
	github.com/docker/docker => github.com/docker/docker v20.10.3-0.20210609100121-ef4d47340142+incompatible
	github.com/hashicorp/go-immutable-radix => github.com/tonistiigi/go-immutable-radix v0.0.0-20170803185627-826af9ccf0fe
)

require (
	github.com/Djarvur/go-err113 v0.1.0 // indirect
	github.com/Microsoft/go-winio v0.5.0 // indirect
	github.com/Microsoft/hcsshim v0.8.17 // indirect
	github.com/aws/aws-sdk-go-v2 v1.6.0
	github.com/aws/aws-sdk-go-v2/config v1.3.0
	github.com/aws/aws-sdk-go-v2/service/ecr v1.3.1
	github.com/containerd/containerd v1.5.2
	github.com/containerd/fuse-overlayfs-snapshotter v1.0.3
	github.com/docker/cli v20.10.7+incompatible
	github.com/docker/distribution v2.7.1+incompatible
	github.com/docker/docker v20.10.7+incompatible
	github.com/go-logr/logr v0.4.0
	github.com/go-logr/zapr v0.4.0
	github.com/golang/groupcache v0.0.0-20210331224755-41bb18bfe9da // indirect
	github.com/golangci/golangci-lint v1.41.1
	github.com/google/gofuzz v1.2.0 // indirect
	github.com/h2non/filetype v1.0.12
	github.com/hashicorp/go-hclog v0.9.2
	github.com/hashicorp/go-plugin v1.3.0
	github.com/mitchellh/mapstructure v1.3.1 // indirect
	github.com/moby/buildkit v0.8.2-0.20210629085500-bb6f11c28d55
	github.com/newrelic/go-agent/v3 v3.9.0
	github.com/newrelic/go-agent/v3/integrations/nrzap v1.0.0
	github.com/opencontainers/go-digest v1.0.0
	github.com/opencontainers/image-spec v1.0.1
	github.com/opencontainers/runc v1.0.0 // indirect
	github.com/pkg/errors v0.9.1
	github.com/prometheus/client_golang v1.11.0
	github.com/prometheus/common v0.29.0 // indirect
	github.com/sirupsen/logrus v1.8.1
	github.com/smartystreets/assertions v1.0.0 // indirect
	github.com/spf13/cast v1.3.1 // indirect
	github.com/spf13/cobra v1.1.3
	github.com/spf13/jwalterweatherman v1.1.0 // indirect
	github.com/streadway/amqp v0.0.0-20200108173154-1c71cc93ed71
	github.com/stretchr/testify v1.7.0
	go.etcd.io/bbolt v1.3.6
	go.opencensus.io v0.23.0 // indirect
	go.uber.org/atomic v1.8.0 // indirect
	go.uber.org/multierr v1.7.0 // indirect
	go.uber.org/zap v1.17.0
	golang.org/x/crypto v0.0.0-20210322153248-0c34fe9e7dc2
	golang.org/x/net v0.0.0-20210614182718-04defd469f4e // indirect
	golang.org/x/oauth2 v0.0.0-20210622215436-a8dc77f794b6 // indirect
	golang.org/x/sync v0.0.0-20210220032951-036812b2e83c
	golang.org/x/sys v0.0.0-20210616094352-59db8d763f22 // indirect
	golang.org/x/term v0.0.0-20210615171337-6886f2dfbf5b // indirect
	google.golang.org/genproto v0.0.0-20210624195500-8bfb893ecb84 // indirect
	google.golang.org/grpc v1.38.0
	gopkg.in/ini.v1 v1.56.0 // indirect
	k8s.io/api v0.21.2
	k8s.io/apiextensions-apiserver v0.21.2
	k8s.io/apimachinery v0.21.2
	k8s.io/client-go v0.21.2
	k8s.io/code-generator v0.21.2
	k8s.io/klog/v2 v2.9.0 // indirect
	k8s.io/kube-openapi v0.0.0-20210527164424-3c818078ee3d // indirect
	k8s.io/utils v0.0.0-20210527160623-6fdb442a123b
	sigs.k8s.io/controller-runtime v0.9.1
	sigs.k8s.io/controller-tools v0.6.1
	sigs.k8s.io/structured-merge-diff/v4 v4.1.1 // indirect
	sigs.k8s.io/yaml v1.2.0
)
