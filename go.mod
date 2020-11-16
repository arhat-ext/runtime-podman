module ext.arhat.dev/runtime-podman

go 1.15

require (
	arhat.dev/aranya-proto v0.2.3
	arhat.dev/arhat-proto v0.4.2
	arhat.dev/libext v0.4.7
	arhat.dev/pkg v0.4.2
	ext.arhat.dev/runtimeutil v0.2.2
	github.com/containers/common v0.26.2
	github.com/containers/image/v5 v5.7.0
	github.com/containers/podman/v2 v2.1.1
	github.com/containers/storage v1.23.6
	github.com/opencontainers/image-spec v1.0.2-0.20190823105129-775207bd45b6
	github.com/opencontainers/runtime-spec v1.0.3-0.20200817204227-f9c09b4ea1df
	github.com/spf13/cobra v1.1.1
	github.com/spf13/pflag v1.0.5
	go.uber.org/multierr v1.6.0
	gopkg.in/yaml.v2 v2.3.0
	k8s.io/client-go v0.19.4
)

// podman v2.1.1
replace (
	github.com/container-storage-interface/spec => github.com/container-storage-interface/spec v1.3.0
	github.com/containerd/containerd => github.com/containerd/containerd v1.3.4
	github.com/containernetworking/cni => github.com/containernetworking/cni v0.8.0
	github.com/containernetworking/plugins => github.com/containernetworking/plugins v0.8.7
	github.com/containers/buildah => github.com/containers/buildah v1.16.1
	github.com/containers/common => github.com/containers/common v0.26.2
	github.com/containers/image/v5 => github.com/containers/image/v5 v5.7.0
	github.com/containers/podman/v2 => github.com/containers/podman/v2 v2.1.1
	github.com/containers/psgo => github.com/containers/psgo v1.5.1
	github.com/containers/storage => github.com/containers/storage v1.23.5
	github.com/coreos/go-systemd/v22 => github.com/coreos/go-systemd/v22 v22.1.0
	github.com/docker/distribution => github.com/docker/distribution v2.7.1+incompatible
	github.com/evanphx/json-patch => github.com/evanphx/json-patch/v5 v5.0.0
	github.com/godbus/dbus/v5 => github.com/godbus/dbus/v5 v5.0.3
	github.com/heketi/heketi => github.com/heketi/heketi v9.0.1-0.20190917153846-c2e2a4ab7ab9+incompatible
	github.com/mindprince/gonvml => github.com/mindprince/gonvml v0.0.0-20190828220739-9ebdce4bb989
	github.com/moby/sys/mountinfo => github.com/moby/sys/mountinfo v0.1.3
	github.com/moby/term => github.com/moby/term v0.0.0-20200915141129-7f0af18e79f2
	github.com/opencontainers/runc => github.com/opencontainers/runc v1.0.0-rc91.0.20200708210054-ce54a9d4d79b
	github.com/opencontainers/runtime-spec => github.com/opencontainers/runtime-spec v1.0.3-0.20200817204227-f9c09b4ea1df
	github.com/opencontainers/runtime-tools => github.com/opencontainers/runtime-tools v0.9.0
	github.com/opencontainers/selinux => github.com/opencontainers/selinux v1.6.0
	github.com/openshift/imagebuilder => github.com/openshift/imagebuilder v1.1.6
	github.com/vishvananda/netlink => github.com/vishvananda/netlink v1.1.0
	go.uber.org/multierr => github.com/uber-go/multierr v1.6.0
)

// go experimental
replace (
	golang.org/x/crypto => github.com/golang/crypto v0.0.0-20200728195943-123391ffb6de
	golang.org/x/exp => github.com/golang/exp v0.0.0-20200513190911-00229845015e
	golang.org/x/lint => github.com/golang/lint v0.0.0-20200302205851-738671d3881b
	golang.org/x/net => github.com/golang/net v0.0.0-20200707034311-ab3426394381
	golang.org/x/oauth2 => github.com/golang/oauth2 v0.0.0-20200107190931-bf48bf16ab8d
	golang.org/x/sync => github.com/golang/sync v0.0.0-20200317015054-43a5402ce75a
	golang.org/x/sys => github.com/golang/sys v0.0.0-20201107080550-4d91cf3a1aaf
	golang.org/x/text => github.com/golang/text v0.3.2
	golang.org/x/tools => github.com/golang/tools v0.0.0-20201023174141-c8cfbd0f21e6
	golang.org/x/xerrors => github.com/golang/xerrors v0.0.0-20200804184101-5ec99f83aff1
)

replace (
	k8s.io/api => github.com/kubernetes/api v0.18.10
	k8s.io/apiextensions-apiserver => github.com/kubernetes/apiextensions-apiserver v0.18.10
	k8s.io/apimachinery => github.com/kubernetes/apimachinery v0.18.10
	k8s.io/apiserver => github.com/kubernetes/apiserver v0.18.10
	k8s.io/cli-runtime => github.com/kubernetes/cli-runtime v0.18.10
	k8s.io/client-go => github.com/kubernetes/client-go v0.18.10
	k8s.io/cloud-provider => github.com/kubernetes/cloud-provider v0.18.10
	k8s.io/cluster-bootstrap => github.com/kubernetes/cluster-bootstrap v0.18.10
	k8s.io/code-generator => github.com/kubernetes/code-generator v0.18.10
	k8s.io/component-base => github.com/kubernetes/component-base v0.18.10
	k8s.io/cri-api => github.com/kubernetes/cri-api v0.18.10
	k8s.io/csi-translation-lib => github.com/kubernetes/csi-translation-lib v0.18.10
	k8s.io/klog => github.com/kubernetes/klog v1.0.0
	k8s.io/klog/v2 => github.com/kubernetes/klog/v2 v2.3.0
	k8s.io/kube-aggregator => github.com/kubernetes/kube-aggregator v0.18.10
	k8s.io/kube-controller-manager => github.com/kubernetes/kube-controller-manager v0.18.10
	k8s.io/kube-proxy => github.com/kubernetes/kube-proxy v0.18.10
	k8s.io/kube-scheduler => github.com/kubernetes/kube-scheduler v0.18.10
	k8s.io/kubectl => github.com/kubernetes/kubectl v0.18.10
	k8s.io/kubelet => github.com/kubernetes/kubelet v0.18.10
	k8s.io/kubernetes => github.com/kubernetes/kubernetes v1.18.10
	k8s.io/legacy-cloud-providers => github.com/kubernetes/legacy-cloud-providers v0.18.10
	k8s.io/metrics => github.com/kubernetes/metrics v0.18.10
	k8s.io/sample-apiserver => github.com/kubernetes/sample-apiserver v0.18.10
	k8s.io/utils => github.com/kubernetes/utils v0.0.0-20200821003339-5e75c0163111
	vbom.ml/util => github.com/fvbommel/util v0.0.2
)

replace (
	cloud.google.com/go => cloud.google.com/go v0.63.0
	github.com/Microsoft/go-winio => github.com/Microsoft/go-winio v0.4.14
	github.com/Microsoft/hcsshim => github.com/Microsoft/hcsshim v0.8.9
	github.com/OpenPeeDeeP/depguard => github.com/OpenPeeDeeP/depguard v1.0.1
	github.com/PuerkitoBio/purell => github.com/PuerkitoBio/purell v1.1.1
	github.com/StackExchange/wmi => github.com/StackExchange/wmi v0.0.0-20190523213315-cbe66965904d
	github.com/alecthomas/participle => github.com/alecthomas/participle v0.5.0
	github.com/alecthomas/units => github.com/alecthomas/units v0.0.0-20190924025748-f65c72e2690d
	github.com/asaskevich/govalidator => github.com/asaskevich/govalidator v0.0.0-20200428143746-21a406dcc535
	github.com/aws/aws-sdk-go => github.com/aws/aws-sdk-go v1.31.7
	github.com/aws/aws-sdk-go-v2 => github.com/aws/aws-sdk-go-v2 v0.23.0
	github.com/containerd/typeurl => github.com/containerd/typeurl v1.0.1
	github.com/creack/pty => github.com/creack/pty v1.1.11
	github.com/docker/docker => github.com/docker/engine v17.12.0-ce-rc1.0.20200618181300-9dc6525e6118+incompatible
	github.com/docker/spdystream => github.com/docker/spdystream v0.0.0-20181023171402-6480d4af844c
	github.com/dsnet/golib => github.com/dsnet/golib v0.0.0-20190531212259-571cdbcff553
	github.com/fsnotify/fsnotify => github.com/fsnotify/fsnotify v1.4.9
	github.com/golang/protobuf => github.com/golang/protobuf v1.4.2
	github.com/google/gofuzz => github.com/google/gofuzz v1.0.0
	github.com/gorilla/mux => github.com/gorilla/mux v1.8.0
	github.com/jmespath/go-jmespath => github.com/jmespath/go-jmespath v0.3.0
	github.com/pion/dtls/v2 => github.com/pion/dtls/v2 v2.0.3
	github.com/spf13/cobra => github.com/spf13/cobra v1.1.1
	github.com/stretchr/testify => github.com/stretchr/testify v1.6.1
	go.etcd.io/etcd => go.etcd.io/etcd v0.5.0-alpha.5.0.20200329194405-dd816f0735f8
	go.uber.org/atomic => go.uber.org/atomic v1.6.0
	go.uber.org/zap => go.uber.org/zap v1.15.0
	google.golang.org/api => google.golang.org/api v0.21.0
	google.golang.org/appengine => google.golang.org/appengine v1.6.6
	google.golang.org/grpc => github.com/grpc/grpc-go v1.33.1
	gopkg.in/yaml.v2 => gopkg.in/yaml.v2 v2.3.0
)

// prometheus related
replace (
	github.com/prometheus-community/windows_exporter => github.com/prometheus-community/windows_exporter v0.15.0
	github.com/prometheus/client_golang => github.com/prometheus/client_golang v1.8.0
	github.com/prometheus/client_model => github.com/prometheus/client_model v0.2.0
	github.com/prometheus/common => github.com/prometheus/common v0.14.0
	github.com/prometheus/node_exporter => github.com/prometheus/node_exporter v1.0.1
	github.com/prometheus/procfs => github.com/prometheus/procfs v0.1.3
	honnef.co/go/tools => honnef.co/go/tools v0.0.1-2020.1.5
)

// azure autorest
replace (
	github.com/Azure/go-autorest => github.com/Azure/go-autorest v14.2.0+incompatible
	github.com/Azure/go-autorest/autorest => github.com/Azure/go-autorest/autorest v0.11.3
	github.com/Azure/go-autorest/autorest/adal => github.com/Azure/go-autorest/autorest/adal v0.9.1
	github.com/Azure/go-autorest/autorest/date => github.com/Azure/go-autorest/autorest/date v0.2.0
	github.com/Azure/go-autorest/autorest/mocks => github.com/Azure/go-autorest/autorest/mocks v0.3.0
	github.com/Azure/go-autorest/autorest/to => github.com/Azure/go-autorest/autorest/to v0.4.0
	github.com/Azure/go-autorest/autorest/validation => github.com/Azure/go-autorest/autorest/validation v0.2.0
)
