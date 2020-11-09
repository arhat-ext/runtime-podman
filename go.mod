module ext.arhat.dev/runtime-podman

go 1.15

require (
	arhat.dev/aranya-proto v0.2.3
	arhat.dev/arhat-proto v0.4.2
	arhat.dev/libext v0.4.1
	arhat.dev/pkg v0.3.1
	ext.arhat.dev/runtimeutil v0.1.2
	github.com/containers/common v0.22.0
	github.com/containers/image/v5 v5.7.0
	github.com/containers/podman/v2 v2.0.0-00010101000000-000000000000
	github.com/containers/storage v1.23.6
	github.com/opencontainers/image-spec v1.0.2-0.20190823105129-775207bd45b6
	github.com/opencontainers/runtime-spec v1.0.3-0.20200817204227-f9c09b4ea1df
	github.com/spf13/cobra v1.1.1
	github.com/spf13/pflag v1.0.5
	go.uber.org/multierr v1.6.0
	gopkg.in/yaml.v2 v2.3.0
	k8s.io/client-go v0.18.10
)

// podman v2.1.1
replace (
	github.com/Microsoft/go-winio => github.com/Microsoft/go-winio v0.4.14
	github.com/container-storage-interface/spec => github.com/container-storage-interface/spec v1.3.0
	github.com/containerd/containerd => github.com/containerd/containerd v1.3.4
	github.com/containernetworking/cni => github.com/containernetworking/cni v0.8.0
	github.com/containernetworking/plugins => github.com/containernetworking/plugins v0.8.7
	github.com/containers/buildah => github.com/containers/buildah v1.16.1
	github.com/containers/common => github.com/containers/common v0.26.2
	github.com/containers/conmon => github.com/containers/conmon v0.4.1-0.20200908203337-35a2fa83022e
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
