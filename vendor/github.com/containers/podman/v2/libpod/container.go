package libpod

import (
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/containernetworking/cni/pkg/types"
	cnitypes "github.com/containernetworking/cni/pkg/types/current"
	"github.com/containers/common/pkg/config"
	"github.com/containers/image/v5/manifest"
	"github.com/containers/podman/v2/libpod/define"
	"github.com/containers/podman/v2/libpod/lock"
	"github.com/containers/podman/v2/pkg/rootless"
	"github.com/containers/podman/v2/utils"
	"github.com/containers/storage"
	"github.com/cri-o/ocicni/pkg/ocicni"
	spec "github.com/opencontainers/runtime-spec/specs-go"
	"github.com/pkg/errors"
)

// CgroupfsDefaultCgroupParent is the cgroup parent for CGroupFS in libpod
const CgroupfsDefaultCgroupParent = "/libpod_parent"

// SystemdDefaultCgroupParent is the cgroup parent for the systemd cgroup
// manager in libpod
const SystemdDefaultCgroupParent = "machine.slice"

// SystemdDefaultRootlessCgroupParent is the cgroup parent for the systemd cgroup
// manager in libpod when running as rootless
const SystemdDefaultRootlessCgroupParent = "user.slice"

// DefaultWaitInterval is the default interval between container status checks
// while waiting.
const DefaultWaitInterval = 250 * time.Millisecond

// LinuxNS represents a Linux namespace
type LinuxNS int

const (
	// InvalidNS is an invalid namespace
	InvalidNS LinuxNS = iota
	// IPCNS is the IPC namespace
	IPCNS LinuxNS = iota
	// MountNS is the mount namespace
	MountNS LinuxNS = iota
	// NetNS is the network namespace
	NetNS LinuxNS = iota
	// PIDNS is the PID namespace
	PIDNS LinuxNS = iota
	// UserNS is the user namespace
	UserNS LinuxNS = iota
	// UTSNS is the UTS namespace
	UTSNS LinuxNS = iota
	// CgroupNS is the CGroup namespace
	CgroupNS LinuxNS = iota
)

// String returns a string representation of a Linux namespace
// It is guaranteed to be the name of the namespace in /proc for valid ns types
func (ns LinuxNS) String() string {
	switch ns {
	case InvalidNS:
		return "invalid"
	case IPCNS:
		return "ipc"
	case MountNS:
		return "mnt"
	case NetNS:
		return "net"
	case PIDNS:
		return "pid"
	case UserNS:
		return "user"
	case UTSNS:
		return "uts"
	case CgroupNS:
		return "cgroup"
	default:
		return "unknown"
	}
}

// Valid restart policy types.
const (
	// RestartPolicyNone indicates that no restart policy has been requested
	// by a container.
	RestartPolicyNone = ""
	// RestartPolicyNo is identical in function to RestartPolicyNone.
	RestartPolicyNo = "no"
	// RestartPolicyAlways unconditionally restarts the container.
	RestartPolicyAlways = "always"
	// RestartPolicyOnFailure restarts the container on non-0 exit code,
	// with an optional maximum number of retries.
	RestartPolicyOnFailure = "on-failure"
	// RestartPolicyUnlessStopped unconditionally restarts unless stopped
	// by the user. It is identical to Always except with respect to
	// handling of system restart, which Podman does not yet support.
	RestartPolicyUnlessStopped = "unless-stopped"
)

// Container is a single OCI container.
// All operations on a Container that access state must begin with a call to
// syncContainer().
// There is no guarantee that state exists in a readable state before
// syncContainer() is run, and even if it does, its contents will be out of date
// and must be refreshed from the database.
// Generally, this requirement applies only to top-level functions; helpers can
// assume that their callers handled this requirement. Generally speaking, if a
// function takes the container lock and accesses any part of state, it should
// syncContainer() immediately after locking.
type Container struct {
	config *ContainerConfig

	state *ContainerState

	// Batched indicates that a container has been locked as part of a
	// Batch() operation
	// Functions called on a batched container will not lock or sync
	batched bool

	valid      bool
	lock       lock.Locker
	runtime    *Runtime
	ociRuntime OCIRuntime

	rootlessSlirpSyncR *os.File
	rootlessSlirpSyncW *os.File

	rootlessPortSyncR *os.File
	rootlessPortSyncW *os.File

	// A restored container should have the same IP address as before
	// being checkpointed. If requestedIP is set it will be used instead
	// of config.StaticIP.
	requestedIP net.IP
	// A restored container should have the same MAC address as before
	// being checkpointed. If requestedMAC is set it will be used instead
	// of config.StaticMAC.
	requestedMAC net.HardwareAddr

	// This is true if a container is restored from a checkpoint.
	restoreFromCheckpoint bool
}

// ContainerState contains the current state of the container
// It is stored on disk in a tmpfs and recreated on reboot
type ContainerState struct {
	// The current state of the running container
	State define.ContainerStatus `json:"state"`
	// The path to the JSON OCI runtime spec for this container
	ConfigPath string `json:"configPath,omitempty"`
	// RunDir is a per-boot directory for container content
	RunDir string `json:"runDir,omitempty"`
	// Mounted indicates whether the container's storage has been mounted
	// for use
	Mounted bool `json:"mounted,omitempty"`
	// Mountpoint contains the path to the container's mounted storage as given
	// by containers/storage.
	Mountpoint string `json:"mountPoint,omitempty"`
	// StartedTime is the time the container was started
	StartedTime time.Time `json:"startedTime,omitempty"`
	// FinishedTime is the time the container finished executing
	FinishedTime time.Time `json:"finishedTime,omitempty"`
	// ExitCode is the exit code returned when the container stopped
	ExitCode int32 `json:"exitCode,omitempty"`
	// Exited is whether the container has exited
	Exited bool `json:"exited,omitempty"`
	// OOMKilled indicates that the container was killed as it ran out of
	// memory
	OOMKilled bool `json:"oomKilled,omitempty"`
	// PID is the PID of a running container
	PID int `json:"pid,omitempty"`
	// ConmonPID is the PID of the container's conmon
	ConmonPID int `json:"conmonPid,omitempty"`
	// ExecSessions contains all exec sessions that are associated with this
	// container.
	ExecSessions map[string]*ExecSession `json:"newExecSessions,omitempty"`
	// LegacyExecSessions are legacy exec sessions from older versions of
	// Podman.
	// These are DEPRECATED and will be removed in a future release.
	LegacyExecSessions map[string]*legacyExecSession `json:"execSessions,omitempty"`
	// NetworkStatus contains the configuration results for all networks
	// the pod is attached to. Only populated if we created a network
	// namespace for the container, and the network namespace is currently
	// active
	NetworkStatus []*cnitypes.Result `json:"networkResults,omitempty"`
	// BindMounts contains files that will be bind-mounted into the
	// container when it is mounted.
	// These include /etc/hosts and /etc/resolv.conf
	// This maps the path the file will be mounted to in the container to
	// the path of the file on disk outside the container
	BindMounts map[string]string `json:"bindMounts,omitempty"`
	// StoppedByUser indicates whether the container was stopped by an
	// explicit call to the Stop() API.
	StoppedByUser bool `json:"stoppedByUser,omitempty"`
	// RestartPolicyMatch indicates whether the conditions for restart
	// policy have been met.
	RestartPolicyMatch bool `json:"restartPolicyMatch,omitempty"`
	// RestartCount is how many times the container was restarted by its
	// restart policy. This is NOT incremented by normal container restarts
	// (only by restart policy).
	RestartCount uint `json:"restartCount,omitempty"`

	// ExtensionStageHooks holds hooks which will be executed by libpod
	// and not delegated to the OCI runtime.
	ExtensionStageHooks map[string][]spec.Hook `json:"extensionStageHooks,omitempty"`

	// containerPlatformState holds platform-specific container state.
	containerPlatformState
}

// ContainerNamedVolume is a named volume that will be mounted into the
// container. Each named volume is a libpod Volume present in the state.
type ContainerNamedVolume struct {
	// Name is the name of the volume to mount in.
	// Must resolve to a valid volume present in this Podman.
	Name string `json:"volumeName"`
	// Dest is the mount's destination
	Dest string `json:"dest"`
	// Options are fstab style mount options
	Options []string `json:"options,omitempty"`
}

// ContainerOverlayVolume is a overlay volume that will be mounted into the
// container. Each volume is a libpod Volume present in the state.
type ContainerOverlayVolume struct {
	// Destination is the absolute path where the mount will be placed in the container.
	Dest string `json:"dest"`
	// Source specifies the source path of the mount.
	Source string `json:"source,omitempty"`
}

// Config accessors
// Unlocked

// Config returns the configuration used to create the container
func (c *Container) Config() *ContainerConfig {
	returnConfig := new(ContainerConfig)
	if err := JSONDeepCopy(c.config, returnConfig); err != nil {
		return nil
	}

	return returnConfig
}

// Spec returns the container's OCI runtime spec
// The spec returned is the one used to create the container. The running
// spec may differ slightly as mounts are added based on the image
func (c *Container) Spec() *spec.Spec {
	returnSpec := new(spec.Spec)
	if err := JSONDeepCopy(c.config.Spec, returnSpec); err != nil {
		return nil
	}

	return returnSpec
}

// specFromState returns the unmarshalled json config of the container.  If the
// config does not exist (e.g., because the container was never started) return
// the spec from the config.
func (c *Container) specFromState() (*spec.Spec, error) {
	returnSpec := c.config.Spec

	if f, err := os.Open(c.state.ConfigPath); err == nil {
		returnSpec = new(spec.Spec)
		content, err := ioutil.ReadAll(f)
		if err != nil {
			return nil, errors.Wrapf(err, "error reading container config")
		}
		if err := json.Unmarshal(content, &returnSpec); err != nil {
			return nil, errors.Wrapf(err, "error unmarshalling container config")
		}
	} else if !os.IsNotExist(err) {
		// ignore when the file does not exist
		return nil, errors.Wrapf(err, "error opening container config")
	}

	return returnSpec, nil
}

// ID returns the container's ID
func (c *Container) ID() string {
	return c.config.ID
}

// Name returns the container's name
func (c *Container) Name() string {
	return c.config.Name
}

// PodID returns the full ID of the pod the container belongs to, or "" if it
// does not belong to a pod
func (c *Container) PodID() string {
	return c.config.Pod
}

// Namespace returns the libpod namespace the container is in.
// Namespaces are used to logically separate containers and pods in the state.
func (c *Container) Namespace() string {
	return c.config.Namespace
}

// Image returns the ID and name of the image used as the container's rootfs.
func (c *Container) Image() (string, string) {
	return c.config.RootfsImageID, c.config.RootfsImageName
}

// RawImageName returns the unprocessed and not-normalized user-specified image
// name.
func (c *Container) RawImageName() string {
	return c.config.RawImageName
}

// ShmDir returns the sources path to be mounted on /dev/shm in container
func (c *Container) ShmDir() string {
	return c.config.ShmDir
}

// ShmSize returns the size of SHM device to be mounted into the container
func (c *Container) ShmSize() int64 {
	return c.config.ShmSize
}

// StaticDir returns the directory used to store persistent container files
func (c *Container) StaticDir() string {
	return c.config.StaticDir
}

// NamedVolumes returns the container's named volumes.
// The name of each is guaranteed to point to a valid libpod Volume present in
// the state.
func (c *Container) NamedVolumes() []*ContainerNamedVolume {
	volumes := []*ContainerNamedVolume{}
	for _, vol := range c.config.NamedVolumes {
		newVol := new(ContainerNamedVolume)
		newVol.Name = vol.Name
		newVol.Dest = vol.Dest
		newVol.Options = vol.Options
		volumes = append(volumes, newVol)
	}

	return volumes
}

// Privileged returns whether the container is privileged
func (c *Container) Privileged() bool {
	return c.config.Privileged
}

// ProcessLabel returns the selinux ProcessLabel of the container
func (c *Container) ProcessLabel() string {
	return c.config.ProcessLabel
}

// MountLabel returns the SELinux mount label of the container
func (c *Container) MountLabel() string {
	return c.config.MountLabel
}

// Systemd returns whether the container will be running in systemd mode
func (c *Container) Systemd() bool {
	return c.config.Systemd
}

// User returns the user who the container is run as
func (c *Container) User() string {
	return c.config.User
}

// Dependencies gets the containers this container depends upon
func (c *Container) Dependencies() []string {
	// Collect in a map first to remove dupes
	dependsCtrs := map[string]bool{}

	// First add all namespace containers
	if c.config.IPCNsCtr != "" {
		dependsCtrs[c.config.IPCNsCtr] = true
	}
	if c.config.MountNsCtr != "" {
		dependsCtrs[c.config.MountNsCtr] = true
	}
	if c.config.NetNsCtr != "" {
		dependsCtrs[c.config.NetNsCtr] = true
	}
	if c.config.PIDNsCtr != "" {
		dependsCtrs[c.config.PIDNsCtr] = true
	}
	if c.config.UserNsCtr != "" {
		dependsCtrs[c.config.UserNsCtr] = true
	}
	if c.config.UTSNsCtr != "" {
		dependsCtrs[c.config.UTSNsCtr] = true
	}
	if c.config.CgroupNsCtr != "" {
		dependsCtrs[c.config.CgroupNsCtr] = true
	}

	// Add all generic dependencies
	for _, id := range c.config.Dependencies {
		dependsCtrs[id] = true
	}

	if len(dependsCtrs) == 0 {
		return []string{}
	}

	depends := make([]string, 0, len(dependsCtrs))
	for ctr := range dependsCtrs {
		depends = append(depends, ctr)
	}

	return depends
}

// NewNetNS returns whether the container will create a new network namespace
func (c *Container) NewNetNS() bool {
	return c.config.CreateNetNS
}

// PortMappings returns the ports that will be mapped into a container if
// a new network namespace is created
// If NewNetNS() is false, this value is unused
func (c *Container) PortMappings() ([]ocicni.PortMapping, error) {
	// First check if the container belongs to a network namespace (like a pod)
	if len(c.config.NetNsCtr) > 0 {
		netNsCtr, err := c.runtime.GetContainer(c.config.NetNsCtr)
		if err != nil {
			return nil, errors.Wrapf(err, "unable to lookup network namespace for container %s", c.ID())
		}
		return netNsCtr.PortMappings()
	}
	return c.config.PortMappings, nil
}

// DNSServers returns DNS servers that will be used in the container's
// resolv.conf
// If empty, DNS server from the host's resolv.conf will be used instead
func (c *Container) DNSServers() []net.IP {
	return c.config.DNSServer
}

// DNSSearch returns the DNS search domains that will be used in the container's
// resolv.conf
// If empty, DNS Search domains from the host's resolv.conf will be used instead
func (c *Container) DNSSearch() []string {
	return c.config.DNSSearch
}

// DNSOption returns the DNS options that will be used in the container's
// resolv.conf
// If empty, options from the host's resolv.conf will be used instead
func (c *Container) DNSOption() []string {
	return c.config.DNSOption
}

// HostsAdd returns hosts that will be added to the container's hosts file
// The host system's hosts file is used as a base, and these are appended to it
func (c *Container) HostsAdd() []string {
	return c.config.HostAdd
}

// UserVolumes returns user-added volume mounts in the container.
// These are not added to the spec, but are used during image commit and to
// trigger some OCI hooks.
func (c *Container) UserVolumes() []string {
	volumes := make([]string, 0, len(c.config.UserVolumes))
	volumes = append(volumes, c.config.UserVolumes...)
	return volumes
}

// Entrypoint is the container's entrypoint.
// This is not added to the spec, but is instead used during image commit.
func (c *Container) Entrypoint() []string {
	entrypoint := make([]string, 0, len(c.config.Entrypoint))
	entrypoint = append(entrypoint, c.config.Entrypoint...)
	return entrypoint
}

// Command is the container's command
// This is not added to the spec, but is instead used during image commit
func (c *Container) Command() []string {
	command := make([]string, 0, len(c.config.Command))
	command = append(command, c.config.Command...)
	return command
}

// Stdin returns whether STDIN on the container will be kept open
func (c *Container) Stdin() bool {
	return c.config.Stdin
}

// Labels returns the container's labels
func (c *Container) Labels() map[string]string {
	labels := make(map[string]string)
	for key, value := range c.config.Labels {
		labels[key] = value
	}
	return labels
}

// StopSignal is the signal that will be used to stop the container
// If it fails to stop the container, SIGKILL will be used after a timeout
// If StopSignal is 0, the default signal of SIGTERM will be used
func (c *Container) StopSignal() uint {
	return c.config.StopSignal
}

// StopTimeout returns the container's stop timeout
// If the container's default stop signal fails to kill the container, SIGKILL
// will be used after this timeout
func (c *Container) StopTimeout() uint {
	return c.config.StopTimeout
}

// CreatedTime gets the time when the container was created
func (c *Container) CreatedTime() time.Time {
	return c.config.CreatedTime
}

// CgroupParent gets the container's CGroup parent
func (c *Container) CgroupParent() string {
	return c.config.CgroupParent
}

// LogPath returns the path to the container's log file
// This file will only be present after Init() is called to create the container
// in the runtime
func (c *Container) LogPath() string {
	return c.config.LogPath
}

// LogTag returns the tag to the container's log file
func (c *Container) LogTag() string {
	return c.config.LogTag
}

// RestartPolicy returns the container's restart policy.
func (c *Container) RestartPolicy() string {
	return c.config.RestartPolicy
}

// RestartRetries returns the number of retries that will be attempted when
// using the "on-failure" restart policy
func (c *Container) RestartRetries() uint {
	return c.config.RestartRetries
}

// LogDriver returns the log driver for this container
func (c *Container) LogDriver() string {
	return c.config.LogDriver
}

// RuntimeName returns the name of the runtime
func (c *Container) RuntimeName() string {
	return c.config.OCIRuntime
}

// Runtime spec accessors
// Unlocked

// Hostname gets the container's hostname
func (c *Container) Hostname() string {
	if c.config.Spec.Hostname != "" {
		return c.config.Spec.Hostname
	}

	if len(c.ID()) < 11 {
		return c.ID()
	}
	return c.ID()[:12]
}

// WorkingDir returns the containers working dir
func (c *Container) WorkingDir() string {
	if c.config.Spec.Process != nil {
		return c.config.Spec.Process.Cwd
	}
	return "/"
}

// State Accessors
// Require locking

// State returns the current state of the container
func (c *Container) State() (define.ContainerStatus, error) {
	if !c.batched {
		c.lock.Lock()
		defer c.lock.Unlock()

		if err := c.syncContainer(); err != nil {
			return define.ContainerStateUnknown, err
		}
	}
	return c.state.State, nil
}

// Mounted returns whether the container is mounted and the path it is mounted
// at (if it is mounted).
// If the container is not mounted, no error is returned, and the mountpoint
// will be set to "".
func (c *Container) Mounted() (bool, string, error) {
	if !c.batched {
		c.lock.Lock()
		defer c.lock.Unlock()
		if err := c.syncContainer(); err != nil {
			return false, "", errors.Wrapf(err, "error updating container %s state", c.ID())
		}
	}
	// We cannot directly return c.state.Mountpoint as it is not guaranteed
	// to be set if the container is mounted, only if the container has been
	// prepared with c.prepare().
	// Instead, let's call into c/storage
	mountedTimes, err := c.runtime.storageService.MountedContainerImage(c.ID())
	if err != nil {
		return false, "", err
	}

	if mountedTimes > 0 {
		mountPoint, err := c.runtime.storageService.GetMountpoint(c.ID())
		if err != nil {
			return false, "", err
		}

		return true, mountPoint, nil
	}

	return false, "", nil
}

// StartedTime is the time the container was started
func (c *Container) StartedTime() (time.Time, error) {
	if !c.batched {
		c.lock.Lock()
		defer c.lock.Unlock()
		if err := c.syncContainer(); err != nil {
			return time.Time{}, errors.Wrapf(err, "error updating container %s state", c.ID())
		}
	}
	return c.state.StartedTime, nil
}

// FinishedTime is the time the container was stopped
func (c *Container) FinishedTime() (time.Time, error) {
	if !c.batched {
		c.lock.Lock()
		defer c.lock.Unlock()
		if err := c.syncContainer(); err != nil {
			return time.Time{}, errors.Wrapf(err, "error updating container %s state", c.ID())
		}
	}
	return c.state.FinishedTime, nil
}

// ExitCode returns the exit code of the container as
// an int32, and whether the container has exited.
// If the container has not exited, exit code will always be 0.
// If the container restarts, the exit code is reset to 0.
func (c *Container) ExitCode() (int32, bool, error) {
	if !c.batched {
		c.lock.Lock()
		defer c.lock.Unlock()
		if err := c.syncContainer(); err != nil {
			return 0, false, errors.Wrapf(err, "error updating container %s state", c.ID())
		}
	}
	return c.state.ExitCode, c.state.Exited, nil
}

// OOMKilled returns whether the container was killed by an OOM condition
func (c *Container) OOMKilled() (bool, error) {
	if !c.batched {
		c.lock.Lock()
		defer c.lock.Unlock()
		if err := c.syncContainer(); err != nil {
			return false, errors.Wrapf(err, "error updating container %s state", c.ID())
		}
	}
	return c.state.OOMKilled, nil
}

// PID returns the PID of the container.
// If the container is not running, a pid of 0 will be returned. No error will
// occur.
func (c *Container) PID() (int, error) {
	if !c.batched {
		c.lock.Lock()
		defer c.lock.Unlock()

		if err := c.syncContainer(); err != nil {
			return -1, err
		}
	}

	return c.state.PID, nil
}

// ConmonPID Returns the PID of the container's conmon process.
// If the container is not running, a PID of 0 will be returned. No error will
// occur.
func (c *Container) ConmonPID() (int, error) {
	if !c.batched {
		c.lock.Lock()
		defer c.lock.Unlock()

		if err := c.syncContainer(); err != nil {
			return -1, err
		}
	}

	return c.state.ConmonPID, nil
}

// ExecSessions retrieves active exec sessions running in the container
func (c *Container) ExecSessions() ([]string, error) {
	if !c.batched {
		c.lock.Lock()
		defer c.lock.Unlock()

		if err := c.syncContainer(); err != nil {
			return nil, err
		}
	}

	ids := make([]string, 0, len(c.state.ExecSessions))
	for id := range c.state.ExecSessions {
		ids = append(ids, id)
	}

	return ids, nil
}

// ExecSession retrieves detailed information on a single active exec session in
// a container
func (c *Container) ExecSession(id string) (*ExecSession, error) {
	if !c.batched {
		c.lock.Lock()
		defer c.lock.Unlock()

		if err := c.syncContainer(); err != nil {
			return nil, err
		}
	}

	session, ok := c.state.ExecSessions[id]
	if !ok {
		return nil, errors.Wrapf(define.ErrNoSuchExecSession, "no exec session with ID %s found in container %s", id, c.ID())
	}

	returnSession := new(ExecSession)
	if err := JSONDeepCopy(session, returnSession); err != nil {
		return nil, errors.Wrapf(err, "error copying contents of container %s exec session %s", c.ID(), session.ID())
	}

	return returnSession, nil
}

// IPs retrieves a container's IP address(es)
// This will only be populated if the container is configured to created a new
// network namespace, and that namespace is presently active
func (c *Container) IPs() ([]net.IPNet, error) {
	if !c.batched {
		c.lock.Lock()
		defer c.lock.Unlock()

		if err := c.syncContainer(); err != nil {
			return nil, err
		}
	}

	if !c.config.CreateNetNS {
		return nil, errors.Wrapf(define.ErrInvalidArg, "container %s network namespace is not managed by libpod", c.ID())
	}

	ips := make([]net.IPNet, 0)

	for _, r := range c.state.NetworkStatus {
		for _, ip := range r.IPs {
			ips = append(ips, ip.Address)
		}
	}

	return ips, nil
}

// Routes retrieves a container's routes
// This will only be populated if the container is configured to created a new
// network namespace, and that namespace is presently active
func (c *Container) Routes() ([]types.Route, error) {
	if !c.batched {
		c.lock.Lock()
		defer c.lock.Unlock()

		if err := c.syncContainer(); err != nil {
			return nil, err
		}
	}

	if !c.config.CreateNetNS {
		return nil, errors.Wrapf(define.ErrInvalidArg, "container %s network namespace is not managed by libpod", c.ID())
	}

	routes := make([]types.Route, 0)

	for _, r := range c.state.NetworkStatus {
		for _, route := range r.Routes {
			newRoute := types.Route{
				Dst: route.Dst,
				GW:  route.GW,
			}
			routes = append(routes, newRoute)
		}
	}

	return routes, nil
}

// BindMounts retrieves bind mounts that were created by libpod and will be
// added to the container
// All these mounts except /dev/shm are ignored if a mount in the given spec has
// the same destination
// These mounts include /etc/resolv.conf, /etc/hosts, and /etc/hostname
// The return is formatted as a map from destination (mountpoint in the
// container) to source (path of the file that will be mounted into the
// container)
// If the container has not been started yet, an empty map will be returned, as
// the files in question are only created when the container is started.
func (c *Container) BindMounts() (map[string]string, error) {
	if !c.batched {
		c.lock.Lock()
		defer c.lock.Unlock()

		if err := c.syncContainer(); err != nil {
			return nil, err
		}
	}

	newMap := make(map[string]string, len(c.state.BindMounts))

	for key, val := range c.state.BindMounts {
		newMap[key] = val
	}

	return newMap, nil
}

// StoppedByUser returns whether the container was last stopped by an explicit
// call to the Stop() API, or whether it exited naturally.
func (c *Container) StoppedByUser() (bool, error) {
	if !c.batched {
		c.lock.Lock()
		defer c.lock.Unlock()

		if err := c.syncContainer(); err != nil {
			return false, err
		}
	}

	return c.state.StoppedByUser, nil
}

// Misc Accessors
// Most will require locking

// NamespacePath returns the path of one of the container's namespaces
// If the container is not running, an error will be returned
func (c *Container) NamespacePath(linuxNS LinuxNS) (string, error) { //nolint:interfacer
	if !c.batched {
		c.lock.Lock()
		defer c.lock.Unlock()
		if err := c.syncContainer(); err != nil {
			return "", errors.Wrapf(err, "error updating container %s state", c.ID())
		}
	}

	if c.state.State != define.ContainerStateRunning && c.state.State != define.ContainerStatePaused {
		return "", errors.Wrapf(define.ErrCtrStopped, "cannot get namespace path unless container %s is running", c.ID())
	}

	if linuxNS == InvalidNS {
		return "", errors.Wrapf(define.ErrInvalidArg, "invalid namespace requested from container %s", c.ID())
	}

	return fmt.Sprintf("/proc/%d/ns/%s", c.state.PID, linuxNS.String()), nil
}

// CGroupPath returns a cgroups "path" for a given container.
func (c *Container) CGroupPath() (string, error) {
	switch {
	case c.config.CgroupsMode == cgroupSplit:
		if c.config.CgroupParent != "" {
			return "", errors.Errorf("cannot specify cgroup-parent with cgroup-mode %q", cgroupSplit)
		}
		cg, err := utils.GetCgroupProcess(c.state.ConmonPID)
		if err != nil {
			return "", err
		}
		// Use the conmon cgroup for two reasons: we validate the container
		// delegation was correct, and the conmon cgroup doesn't change at runtime
		// while we are not sure about the container that can create sub cgroups.
		if !strings.HasSuffix(cg, "supervisor") {
			return "", errors.Errorf("invalid cgroup for conmon %q", cg)
		}
		return strings.TrimSuffix(cg, "/supervisor") + "/container", nil
	case c.runtime.config.Engine.CgroupManager == config.CgroupfsCgroupsManager:
		return filepath.Join(c.config.CgroupParent, fmt.Sprintf("libpod-%s", c.ID())), nil
	case c.runtime.config.Engine.CgroupManager == config.SystemdCgroupsManager:
		if rootless.IsRootless() {
			uid := rootless.GetRootlessUID()
			parts := strings.SplitN(c.config.CgroupParent, "/", 2)

			dir := ""
			if len(parts) > 1 {
				dir = parts[1]
			}

			return filepath.Join(parts[0], fmt.Sprintf("user-%d.slice/user@%d.service/user.slice/%s", uid, uid, dir), createUnitName("libpod", c.ID())), nil
		}
		return filepath.Join(c.config.CgroupParent, createUnitName("libpod", c.ID())), nil
	default:
		return "", errors.Wrapf(define.ErrInvalidArg, "unsupported CGroup manager %s in use", c.runtime.config.Engine.CgroupManager)
	}
}

// RootFsSize returns the root FS size of the container
func (c *Container) RootFsSize() (int64, error) {
	if !c.batched {
		c.lock.Lock()
		defer c.lock.Unlock()
		if err := c.syncContainer(); err != nil {
			return -1, errors.Wrapf(err, "error updating container %s state", c.ID())
		}
	}
	return c.rootFsSize()
}

// RWSize returns the rw size of the container
func (c *Container) RWSize() (int64, error) {
	if !c.batched {
		c.lock.Lock()
		defer c.lock.Unlock()
		if err := c.syncContainer(); err != nil {
			return -1, errors.Wrapf(err, "error updating container %s state", c.ID())
		}
	}
	return c.rwSize()
}

// IDMappings returns the UID/GID mapping used for the container
func (c *Container) IDMappings() (storage.IDMappingOptions, error) {
	return c.config.IDMappings, nil
}

// RootUID returns the root user mapping from container
func (c *Container) RootUID() int {
	for _, uidmap := range c.config.IDMappings.UIDMap {
		if uidmap.ContainerID == 0 {
			return uidmap.HostID
		}
	}
	return 0
}

// RootGID returns the root user mapping from container
func (c *Container) RootGID() int {
	for _, gidmap := range c.config.IDMappings.GIDMap {
		if gidmap.ContainerID == 0 {
			return gidmap.HostID
		}
	}
	return 0
}

// IsInfra returns whether the container is an infra container
func (c *Container) IsInfra() bool {
	return c.config.IsInfra
}

// IsReadOnly returns whether the container is running in read only mode
func (c *Container) IsReadOnly() bool {
	return c.config.Spec.Root.Readonly
}

// NetworkDisabled returns whether the container is running with a disabled network
func (c *Container) NetworkDisabled() (bool, error) {
	if c.config.NetNsCtr != "" {
		container, err := c.runtime.state.Container(c.config.NetNsCtr)
		if err != nil {
			return false, err
		}
		return container.NetworkDisabled()
	}
	return networkDisabled(c)

}

func networkDisabled(c *Container) (bool, error) {
	if c.config.CreateNetNS {
		return false, nil
	}
	if !c.config.PostConfigureNetNS {
		for _, ns := range c.config.Spec.Linux.Namespaces {
			if ns.Type == spec.NetworkNamespace {
				return ns.Path == "", nil
			}
		}
	}
	return false, nil
}

// ContainerState returns containerstate struct
func (c *Container) ContainerState() (*ContainerState, error) {
	if !c.batched {
		c.lock.Lock()
		defer c.lock.Unlock()

		if err := c.syncContainer(); err != nil {
			return nil, err
		}
	}
	returnConfig := new(ContainerState)
	if err := JSONDeepCopy(c.state, returnConfig); err != nil {
		return nil, errors.Wrapf(err, "error copying container %s state", c.ID())
	}
	return c.state, nil
}

// HasHealthCheck returns bool as to whether there is a health check
// defined for the container
func (c *Container) HasHealthCheck() bool {
	return c.config.HealthCheckConfig != nil
}

// HealthCheckConfig returns the command and timing attributes of the health check
func (c *Container) HealthCheckConfig() *manifest.Schema2HealthConfig {
	return c.config.HealthCheckConfig
}

// AutoRemove indicates whether the container will be removed after it is executed
func (c *Container) AutoRemove() bool {
	spec := c.config.Spec
	if spec.Annotations == nil {
		return false
	}
	return c.Spec().Annotations[define.InspectAnnotationAutoremove] == define.InspectResponseTrue
}

// Timezone returns the timezone configured inside the container.
// Local means it has the same timezone as the host machine
func (c *Container) Timezone() string {
	return c.config.Timezone
}

// Umask returns the Umask bits configured inside the container.
func (c *Container) Umask() string {
	return c.config.Umask
}