/*
Copyright 2020 The arhat.dev Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package runtime

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"arhat.dev/aranya-proto/aranyagopb"
	"arhat.dev/aranya-proto/aranyagopb/runtimepb"
	"arhat.dev/libext/types"
	"arhat.dev/pkg/log"
	"arhat.dev/pkg/wellknownerrors"
	"ext.arhat.dev/runtimeutil/containerutil"
	"ext.arhat.dev/runtimeutil/storageutil"
	"github.com/containers/image/v5/manifest"
	"github.com/containers/podman/v2/libpod"
	"github.com/containers/podman/v2/libpod/define"
	libpodns "github.com/containers/podman/v2/pkg/namespaces"
	libpodspec "github.com/containers/podman/v2/pkg/spec"
	ctrstorage "github.com/containers/storage"
	ociruntimespec "github.com/opencontainers/runtime-spec/specs-go"
	"go.uber.org/multierr"
	"k8s.io/client-go/tools/remotecommand"
)

func (r *libpodRuntime) listContainersByLabels(labels map[string]string) ([]*libpod.Container, error) {
	containers, err := r.runtimeClient.GetContainers(containerLabelFilterFunc(labels))
	if err != nil {
		return nil, err
	}

	return containers, nil
}

func (r *libpodRuntime) listPauseContainers() ([]*libpod.Container, error) {
	return r.listContainersByLabels(
		map[string]string{
			containerutil.ContainerLabelPodContainerRole: containerutil.ContainerRoleInfra,
		},
	)
}

func (r *libpodRuntime) findContainerByLabels(labels map[string]string) (*libpod.Container, error) {
	containers, err := r.listContainersByLabels(labels)
	if err != nil {
		return nil, err
	}

	if len(containers) == 0 {
		return nil, wellknownerrors.ErrNotFound
	}

	return containers[0], nil
}

func (r *libpodRuntime) findAbbotContainer() (*libpod.Container, error) {
	return r.findContainerByLabels(containerutil.AbbotMatchLabels())
}

func (r *libpodRuntime) findContainer(podUID, container string) (*libpod.Container, error) {
	return r.findContainerByLabels(map[string]string{
		containerutil.ContainerLabelPodUID:       podUID,
		containerutil.ContainerLabelPodContainer: container,
	})
}

func (r *libpodRuntime) findPod(podUID string) (*libpod.Pod, error) {
	pods, err := r.runtimeClient.Pods(podLabelFilterFunc(map[string]string{
		containerutil.ContainerLabelPodUID: podUID,
	}))
	if err != nil {
		return nil, err
	}

	switch len(pods) {
	case 0:
		return nil, wellknownerrors.ErrNotFound
	case 1:
		return pods[0], nil
	default:
		return nil, fmt.Errorf("unexpected multiple pods for single pod uid")
	}
}

// nolint:unused
func (r *libpodRuntime) startPod(
	logger log.Interface,
	ctx context.Context,
	pod *libpod.Pod,
	containers map[*libpod.Container]*runtimepb.ContainerAction,
) error {
	errMap, err := pod.Start(ctx)
	if err != nil {
		var errList []error
		for id, err := range errMap {
			errList = append(errList, err)
			logger.I("failed to start container", log.String("containerID", id), log.Error(err))
		}
		if len(errList) != 0 {
			return multierr.Combine(errList...)
		}

		return err
	}

	for ctr, postStartHook := range containers {
		if postStartHook != nil {
			logger.D("executing post-start hook")
			if err := r.doHookAction(ctx, ctr, postStartHook); err != nil {
				logger.I("failed to execute post-start hook", log.Error(err))
			}
		}
	}

	return nil
}

func (r *libpodRuntime) execInContainer(
	ctx context.Context,
	ctr *libpod.Container,
	stdin io.Reader,
	stdout, stderr io.Writer,
	command []string,
	tty bool,
	env map[string]string,
) (
	resizeFunc types.ResizeHandleFunc,
	_ <-chan *aranyagopb.ErrorMsg,
	err error,
) {
	streams := r.translateStreams(stdin, stdout, stderr)

	resize := make(chan remotecommand.TerminalSize)
	execConfig := &libpod.ExecConfig{
		Command:      command,
		Terminal:     tty,
		AttachStdin:  streams.AttachInput,
		AttachStdout: streams.AttachOutput,
		AttachStderr: streams.AttachError,
		Environment:  env,
	}

	errCh := make(chan *aranyagopb.ErrorMsg, 1)
	go func() {
		defer func() {
			close(errCh)
		}()

		exitCode, err := ctr.Exec(execConfig, streams, resize)
		if err != nil {
			select {
			case <-ctx.Done():
			case errCh <- &aranyagopb.ErrorMsg{
				Kind:        aranyagopb.ERR_COMMON,
				Description: err.Error(),
				Code:        int64(exitCode),
			}:
			}
		}
	}()

	return func(cols, rows uint32) {
		select {
		case <-ctx.Done():
		case resize <- remotecommand.TerminalSize{Width: uint16(cols), Height: uint16(rows)}:
		}
	}, errCh, nil
}

func (r *libpodRuntime) createPauseContainer(
	ctx context.Context,
	options *runtimepb.PodEnsureCmd,
) (_ *libpod.Pod, _ *libpod.Container, abbotRespBytes []byte, err error) {
	logger := r.logger.WithFields(log.String("action", "createPauseContainer"))
	podName := fmt.Sprintf("%s.%s", options.Namespace, options.Name)

	logger.V("looking up previously create pod", log.String("name", podName))
	prevPod, _ := r.runtimeClient.LookupPod(podName)
	if prevPod != nil {
		logger.V("found previously created pod, deleting", log.String("name", podName))
		if err = r.deletePod(ctx, prevPod); err != nil {
			logger.I("failed to delete previously created pod", log.Error(err))
			return nil, nil, nil, err
		}
	}

	// refuse to create pod using cluster network if no abbot found
	if !options.HostNetwork {
		_, err = r.findAbbotContainer()
		if err != nil {
			return nil, nil, nil, fmt.Errorf("abbot container required but not found: %w", err)
		}
	}

	var hosts []string
	image, err := r.imageClient.NewFromLocal(r.pauseImage)
	if err != nil {
		return nil, nil, nil, err
	}

	imageConfig, err := r.getImageConfig(ctx, image)
	if err != nil {
		return nil, nil, nil, err
	}

	podLabels := containerutil.ContainerLabels(options, containerutil.ContainerNamePause)
	delete(podLabels, containerutil.ContainerLabelPodContainer)
	delete(podLabels, containerutil.ContainerLabelPodContainerRole)

	podOpts := []libpod.PodCreateOption{
		libpod.WithPodName(podName),
		libpod.WithPodLabels(podLabels),
	}

	logger.D("creating new pod", log.String("name", podName))
	pod, err := r.runtimeClient.NewPod(ctx, podOpts...)
	if err != nil {
		logger.I("failed to create new pod", log.Error(err))
		return nil, nil, nil, err
	}
	defer func() {
		if err != nil {
			logger.D("deleting pod due to error", log.NamedError("reason", err))
			if err2 := r.deletePod(context.TODO(), pod); err2 != nil {
				logger.I("failed to delete pod when error happened", log.Error(err2))
			}
		}
	}()

	workDir := "/"
	if imageConfig.WorkingDir != "" {
		workDir = imageConfig.WorkingDir
	}

	config := &libpodspec.CreateConfig{
		Pod:           pod.ID(),
		Env:           containerutil.GetEnv(imageConfig.Env),
		Name:          containerutil.GetContainerName(options.Namespace, options.Name, containerutil.ContainerNamePause),
		Labels:        containerutil.ContainerLabels(options, containerutil.ContainerNamePause),
		Entrypoint:    r.pauseCommand,
		Command:       nil,
		LogDriver:     define.KubernetesLogging,
		WorkDir:       workDir,
		Image:         r.pauseImage,
		ImageID:       image.ID(),
		StopSignal:    syscall.SIGTERM,
		RestartPolicy: r.translateRestartPolicy(runtimepb.RESTART_ALWAYS),

		Security: libpodspec.SecurityConfig{
			// TODO: apply seccomp profile when kubernetes make it GA
			SeccompProfilePath: "unconfined",
		},
		Uts: libpodspec.UtsConfig{
			UtsMode:  "",
			HostAdd:  hosts,
			Hostname: options.Name,
		},
		User: libpodspec.UserConfig{
			IDMappings: &ctrstorage.IDMappingOptions{},
			// TODO: set user namespace properly,
			// 		 host userns is a work around in libpod 1.5.0+
			// nolint:goconst
			UsernsMode: "host",
		},
		Ipc: libpodspec.IpcConfig{
			IpcMode: func() libpodns.IpcMode {
				if options.HostIpc {
					// nolint:goconst
					return "host"
				}
				return "shareable"
			}(),
		},
		Network: libpodspec.NetworkConfig{
			NetMode: func() libpodns.NetworkMode {
				if options.HostNetwork {
					// nolint:goconst
					return "host"
				}
				// do not use libpod default network "default" or ""
				return "none"
			}(),
		},
		Pid: libpodspec.PidConfig{
			PidMode: func() libpodns.PidMode {
				if options.HostPid {
					// nolint:goconst
					return "host"
				}
				return ""
			}(),
		},
	}

	runtimeSpec, opts, err := config.MakeContainerConfig(r.runtimeClient, pod)
	if err != nil {
		return nil, nil, nil, err
	}

	pauseCtr, err := r.runtimeClient.NewContainer(ctx, runtimeSpec, opts...)
	if err != nil {
		return nil, nil, nil, err
	}
	defer func() {
		if err != nil {
			logger.D("deleting pause container due to error", log.NamedError("reason", err))
			if err2 := r.deletePod(context.TODO(), pod); err2 != nil {
				logger.I("failed to delete pause container when error happened", log.Error(err2))
			}
		}
	}()

	logger.D("starting pause container")
	err = pauseCtr.Start(ctx, true)
	if err != nil {
		logger.I("failed to start pause container", log.Error(err))
		return nil, nil, nil, err
	}

	logger.D("checking pause container running")
	code, exited, err := pauseCtr.ExitCode()
	if err != nil {
		logger.I("failed to check pause container running", log.Error(err))
		return nil, nil, nil, err
	}

	if exited {
		logger.I("pause container exited", log.Int32("code", code))
		return nil, nil, nil, fmt.Errorf("pause container exited with code %d", code)
	}

	if !options.HostNetwork {
		pid, _ := pauseCtr.PID()
		abbotRespBytes, err = r.networkClient.Do(
			ctx,
			options.Network.AbbotRequestBytes,
			int64(pid),
			pauseCtr.ID(),
		)
		if err != nil {
			return nil, nil, nil, err
		}
	}

	return pod, pauseCtr, abbotRespBytes, nil
}

// nolint:gocyclo
func (r *libpodRuntime) createContainer(
	ctx context.Context,
	options *runtimepb.PodEnsureCmd,
	pod *libpod.Pod,
	spec *runtimepb.ContainerSpec,
	ns map[string]string,
) (*libpod.Container, error) {
	var (
		useInit       bool
		useSystemd    bool
		userAndGroup  string
		mounts        []ociruntimespec.Mount
		hostPaths     = options.GetVolumes().GetHostPaths()
		volumeData    = options.GetVolumes().GetVolumeData()
		containerName = containerutil.GetContainerName(options.Namespace, options.Name, spec.Name)
		healthCheck   *manifest.Schema2HealthConfig
		entrypoint    = spec.Command
		command       []string
		hostname      string
	)

	switch {
	case options.HostNetwork:
		hostname = ""
	case options.Hostname != "":
		hostname = options.Hostname
	default:
		hostname = options.Name
	}

	// generalize to avoid panic
	if hostPaths == nil {
		hostPaths = make(map[string]string)
	}

	if volumeData == nil {
		volumeData = make(map[string]*runtimepb.NamedData)
	}

	for volName, volMountSpec := range spec.Mounts {
		var source string

		// check if it is host volume, emptyDir, or remote volume
		hostPath, isHostVol := hostPaths[volName]
		if isHostVol {
			var err error
			source, err = storageutil.ResolveHostPathMountSource(
				hostPath, options.PodUid, volName,
				volMountSpec.Remote,
				r.PodRemoteVolumeDir, r.PodTmpfsVolumeDir,
			)
			if err != nil {
				return nil, err
			}

			if volMountSpec.Remote {
				// for remote volume, hostPath is the aranya pod host path
				err := r.storageClient.Mount(context.TODO(), hostPath, source, r.handleStorageFailure(options.PodUid))
				if err != nil {
					return nil, err
				}
			}
		}

		// check if it is vol data (from configMap, Secret)
		if volData, isVolData := volumeData[volName]; isVolData {
			if dataMap := volData.GetDataMap(); dataMap != nil {
				dir := r.PodBindVolumeDir(options.PodUid, volName)
				err := os.MkdirAll(dir, 0750)
				if err != nil {
					return nil, err
				}

				source, err = volMountSpec.Ensure(dir, dataMap)
				if err != nil {
					return nil, err
				}
			}
		}

		options := make([]string, len(volMountSpec.Options))
		_ = copy(options, volMountSpec.Options)

		if volMountSpec.ReadOnly {
			options = append(options, "ro")
		}
		mounts = append(mounts, ociruntimespec.Mount{
			Type:        "bind",
			Source:      source,
			Destination: filepath.Join(volMountSpec.MountPath, volMountSpec.SubPath),
			Options:     options,
		})
	}

	if netOpts := options.Network; len(netOpts.Nameservers) != 0 {
		resolvConfFile := r.PodResolvConfFile(options.PodUid)
		if err := os.MkdirAll(filepath.Dir(resolvConfFile), 0750); err != nil {
			return nil, err
		}

		data, err := r.networkClient.CreateResolvConf(netOpts.Nameservers, netOpts.DnsSearches, netOpts.DnsOptions)
		if err != nil {
			return nil, err
		}

		if err = ioutil.WriteFile(resolvConfFile, data, 0440); err != nil {
			return nil, err
		}

		mounts = append(mounts, ociruntimespec.Mount{
			Type:        "bind",
			Source:      resolvConfFile,
			Destination: "/etc/resolv.conf",
		})
	}

	if probe := spec.LivenessCheck; probe != nil && probe.Method != nil {
		switch action := spec.LivenessCheck.Method.Action.(type) {
		case *runtimepb.ContainerAction_Exec_:
			healthCheck = &manifest.Schema2HealthConfig{
				Test:        append([]string{"CMD"}, action.Exec.Command...),
				Interval:    time.Duration(probe.ProbeInterval),
				Timeout:     time.Duration(probe.ProbeTimeout),
				StartPeriod: time.Duration(probe.InitialDelay),
				Retries:     int(probe.FailureThreshold),
				// TODO: implement success threshold
			}
		case *runtimepb.ContainerAction_Socket_:
			// TODO: implement
		case *runtimepb.ContainerAction_Http:
			// TODO: implement
		}
	}

	image, err := r.imageClient.NewFromLocal(spec.Image)
	if err != nil {
		return nil, err
	}

	imageConfig, err := r.getImageConfig(ctx, image)
	if err != nil {
		return nil, err
	}

	if spec.Security != nil {
		builder := new(strings.Builder)
		if uid := spec.Security.GetUser(); uid != -1 {
			builder.WriteString(strconv.FormatInt(uid, 10))
		}

		if gid := spec.Security.GetGroup(); gid != -1 {
			builder.WriteString(":")
			builder.WriteString(strconv.FormatInt(gid, 10))
		}
		userAndGroup = builder.String()
	}

	if userAndGroup == "" {
		userAndGroup = imageConfig.User
	}

	// workDir MUST NOT be empty
	workDir := spec.WorkingDir
	if workDir == "" {
		if imageConfig.WorkingDir != "" {
			workDir = imageConfig.WorkingDir
		} else {
			workDir = "/"
		}
	}

	var useDefaultEntrypoint bool
	if len(entrypoint) == 0 {
		useDefaultEntrypoint = true
		if len(imageConfig.Entrypoint) > 0 {
			entrypoint = imageConfig.Entrypoint
		} else {
			entrypoint = []string{"/bin/sh", "-c"}
		}
	}

	command = append([]string{}, entrypoint...)
	if len(spec.Args) > 0 {
		command = append(command, spec.Args...)
	} else if useDefaultEntrypoint {
		// use image defaults
		command = append(command, imageConfig.Cmd...)
	}

	// add env from image config
	env := containerutil.GetEnv(imageConfig.Env)
	// add env from pod spec
	for k, v := range spec.Envs {
		env[k] = v
	}

	switch {
	case entrypoint[0] == "/sbin/init", entrypoint[0] == "/usr/sbin/init":
		useInit = true
	case filepath.Base(entrypoint[0]) == "systemd":
		useSystemd = true
	}

	var securityOpts []string
	if opts := spec.Security.GetSelinuxOptions(); opts != nil {
		if opts.User != "" {
			securityOpts = append(securityOpts, fmt.Sprintf("user:%s", opts.User))
		}
		if opts.Type != "" {
			securityOpts = append(securityOpts, fmt.Sprintf("type:%s", opts.Type))
		}
		if opts.Level != "" {
			securityOpts = append(securityOpts, fmt.Sprintf("level:%s", opts.Level))
		}
		if opts.Role != "" {
			securityOpts = append(securityOpts, fmt.Sprintf("role:%s", opts.Role))
		}
	}

	createConfig := &libpodspec.CreateConfig{
		Pod: pod.ID(),
		Tty: spec.Tty,
		Env: env,

		LogDriver:  define.KubernetesLogging,
		Entrypoint: entrypoint,
		Command:    command,

		HealthCheck: healthCheck,
		Image:       spec.Image,
		ImageID:     image.ID(),
		WorkDir:     workDir,

		Labels:     containerutil.ContainerLabels(options, spec.Name),
		StopSignal: syscall.SIGTERM,

		Name:    containerName,
		Systemd: useSystemd,
		Init:    useInit,

		// volume mounts
		Mounts: mounts,

		RestartPolicy: r.translateRestartPolicy(options.RestartPolicy),
		// share namespaces
		Network: libpodspec.NetworkConfig{
			NetMode: libpodns.NetworkMode(ns["net"]),
		},
		Ipc: libpodspec.IpcConfig{
			IpcMode: libpodns.IpcMode(ns["ipc"]),
		},
		Uts: libpodspec.UtsConfig{
			UtsMode:  libpodns.UTSMode(ns["uts"]),
			Hostname: hostname,
		},
		Pid: libpodspec.PidConfig{
			PidMode: libpodns.PidMode(ns["pid"]),
		},
		User: libpodspec.UserConfig{
			// TODO: set userns properly,
			// 		 host userns is a work around in libpod 1.5.0+
			GroupAdd: nil,
			IDMappings: &ctrstorage.IDMappingOptions{
				HostGIDMapping: true,
				HostUIDMapping: true,
			},
			UsernsMode: "host",
			User:       userAndGroup,
		},

		Security: libpodspec.SecurityConfig{
			Privileged:     spec.Security.GetPrivileged(),
			CapAdd:         spec.Security.GetCapsAdd(),
			CapDrop:        spec.Security.GetCapsDrop(),
			ReadOnlyRootfs: spec.Security.GetReadOnlyRootfs(),
			NoNewPrivs:     !spec.Security.GetAllowNewPrivileges(),
			SecurityOpts:   securityOpts,

			LabelOpts:       nil,
			ApparmorProfile: "",
			// TODO: apply seccomp profile when kubernetes make it GA
			SeccompProfilePath: "unconfined",
			ReadOnlyTmpfs:      false,
			Sysctl:             options.GetSecurity().GetSysctls(),
		},

		// security options

		Resources: libpodspec.CreateResourceConfig{
			// TODO: implement
		},
	}

	runtimeSpec, opts, err := createConfig.MakeContainerConfig(r.runtimeClient, pod)
	if err != nil {
		return nil, err
	}

	if spec.Stdin {
		opts = append(opts, libpod.WithStdin())
	}

	ctr, err := r.runtimeClient.NewContainer(ctx, runtimeSpec, opts...)
	if err != nil {
		return nil, err
	}

	return ctr, nil
}

func (r *libpodRuntime) deleteContainer(ctr *libpod.Container) error {
	labels := ctr.Labels()

	// deny abbot pod deletion if having cluster network containers
	if containerutil.IsAbbotPod(labels) {
		containers, err := r.listPauseContainers()
		if err != nil {
			return err
		}

		for _, ctr := range containers {
			if !containerutil.IsHostNetwork(ctr.Labels()) {
				return fmt.Errorf("unable to delete abbot container: %w", wellknownerrors.ErrInvalidOperation)
			}
		}
	}

	_, ok := labels[containerutil.ContainerLabelPodUID]
	if ok {
		// not managed by us
		return nil
	}

	if name, ok := labels[containerutil.ContainerLabelPodContainer]; !ok {
		// not managed by us
		return nil
	} else if name == containerutil.ContainerNamePause {
		// is pause container, deny, delete pod instead
		return fmt.Errorf(
			"pause container should not be delete as normal container: %w",
			wellknownerrors.ErrInvalidOperation,
		)
	}

	_ = ctr.Stop()
	return r.runtimeClient.RemoveContainer(context.TODO(), ctr, true, true)
}

func (r *libpodRuntime) deletePod(ctx context.Context, pod *libpod.Pod) error {
	// delete pod network (best effort)
	podUID, ok := pod.Labels()[containerutil.ContainerLabelPodUID]
	if ok {
		// valid pod, try to delete network
		pauseCtr, err := r.findContainer(podUID, containerutil.ContainerNamePause)
		if err != nil {
			return err
		}

		_ = pauseCtr.Sync()

		labels := pauseCtr.Labels()
		if !containerutil.IsHostNetwork(labels) {
			pid, _ := pauseCtr.PID()
			if err = r.networkClient.Delete(ctx, int64(pid), pauseCtr.ID()); err != nil {
				// refuse to delete pod if network not deleted
				return err
			}
		}

		// deny abbot pod deletion if having cluster network containers
		if containerutil.IsAbbotPod(labels) {
			containers, err := r.listPauseContainers()
			if err != nil {
				return err
			}

			for _, ctr := range containers {
				if !containerutil.IsHostNetwork(ctr.Labels()) {
					return fmt.Errorf("unable to delete abbot container: %w", wellknownerrors.ErrInvalidOperation)
				}
			}
		}
	}

	return r.runtimeClient.RemovePod(context.Background(), pod, true, true)
}

func (r *libpodRuntime) handleStorageFailure(podUID string) storageutil.ExitHandleFunc {
	logger := r.logger.WithFields(log.String("module", "storage"), log.String("podUID", podUID))
	return func(remotePath, mountPoint string, err error) {
		if err != nil {
			logger.I("storage mounter exited", log.Error(err))
		}

		_, e := r.findContainer(podUID, containerutil.ContainerNamePause)
		if errors.Is(e, wellknownerrors.ErrNotFound) {
			logger.D("pod not found, no more remount action")
			return
		}

		err = r.storageClient.Mount(context.TODO(), remotePath, mountPoint, r.handleStorageFailure(podUID))
		if err != nil {
			logger.I("failed to mount remote volume", log.Error(err))
		}
	}
}
