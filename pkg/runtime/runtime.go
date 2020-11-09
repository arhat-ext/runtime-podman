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
	"os"
	"runtime"
	"strings"
	"time"

	"arhat.dev/aranya-proto/aranyagopb"
	"arhat.dev/aranya-proto/aranyagopb/runtimepb"
	"arhat.dev/libext/extruntime"
	"arhat.dev/libext/types"
	"arhat.dev/pkg/log"
	"arhat.dev/pkg/wellknownerrors"
	"ext.arhat.dev/runtimeutil"
	"ext.arhat.dev/runtimeutil/storage"
	libpodconfig "github.com/containers/common/pkg/config"
	"github.com/containers/podman/v2/libpod"
	"github.com/containers/podman/v2/libpod/define"
	libpodimage "github.com/containers/podman/v2/libpod/image"
	libpodversion "github.com/containers/podman/v2/version"
	"k8s.io/client-go/tools/remotecommand"

	"ext.arhat.dev/runtime-podman/pkg/conf"
	"ext.arhat.dev/runtime-podman/pkg/constant"
	"ext.arhat.dev/runtime-podman/pkg/version"
)

func NewPodmanRuntime(
	ctx context.Context,
	logger log.Interface,
	storage *storage.Client,
	config *conf.RuntimeConfig,
) (extruntime.RuntimeEngine, error) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	pauseCmd := constant.DefaultPauseCommand
	if len(config.PauseCommand) > 0 {
		pauseCmd = config.PauseCommand[0]
	}

	opts := []libpod.RuntimeOption{
		libpod.WithNamespace(config.ManagementNamespace),
		libpod.WithDefaultInfraImage(config.PauseImage),
		libpod.WithDefaultInfraCommand(pauseCmd),
		libpod.WithDefaultTransport("docker://"),
	}

	var (
		runtimeClient *libpod.Runtime
		err           error
	)

	// path to the libpod.conf
	configFile, ok := os.LookupEnv("ARHAT_LIBPOD_CONFIG")
	if ok && configFile != "" {
		var rtConfig *libpodconfig.Config
		rtConfig, err = libpodconfig.NewConfig(configFile)
		if err != nil {
			return nil, fmt.Errorf("failed to load custom libpod config: %w", err)
		}
		runtimeClient, err = libpod.NewRuntimeFromConfig(ctx, rtConfig)
	} else {
		// try to create runtime with default options
		runtimeClient, err = libpod.NewRuntime(ctx, opts...)
	}
	if err != nil {
		return nil, err
	}

	imageClient := runtimeClient.ImageRuntime()
	if imageClient == nil {
		return nil, errors.New("empty image client")
	}

	rt := &libpodRuntime{
		logger: logger,
		ctx:    ctx,

		BaseRuntime: runtimeutil.NewBaseRuntime(
			ctx, config.DataDir, config.ImageActionTimeout, config.PodActionTimeout,
			"podman", libpodversion.Version.String(),
			runtime.GOOS, "", version.Arch(), "",
		),

		pauseImage:   config.PauseImage,
		pauseCommand: config.PauseCommand,

		runtimeClient: runtimeClient,
		imageClient:   imageClient,
		storageClient: storage,

		networkClient: nil,
	}

	rt.networkClient = runtimeutil.NewNetworkClient(rt.handleAbbotExec)

	return rt, nil
}

type libpodRuntime struct {
	logger log.Interface

	ctx context.Context
	*runtimeutil.BaseRuntime

	pauseImage   string
	pauseCommand []string
	abbotSubCmd  string

	runtimeClient *libpod.Runtime
	imageClient   *libpodimage.Runtime
	storageClient *storage.Client

	networkClient *runtimeutil.NetworkClient
}

func (r *libpodRuntime) handleAbbotExec(ctx context.Context, env map[string]string, stdin io.Reader, stdout, stderr io.Writer) error {
	logger := r.logger.WithName("network")
	ctr, err := r.findAbbotContainer()
	if err != nil {
		return err
	}

	// try to start abbot container if stopped
	logger.D("checking abbot status")
	status, err := ctr.State()
	if err != nil {
		return fmt.Errorf("failed to get abbot contaienr status: %w", err)
	}

	if status != define.ContainerStateRunning {
		logger.D("abbot not running, trying to start", log.String("status", status.String()))
		err = ctr.Start(ctx, true)
		if err != nil {
			return fmt.Errorf("failed to start abbot container: %w", err)
		}

		time.Sleep(5 * time.Second)
		logger.D("check abbot status again")
		status, err = ctr.State()
		if err != nil {
			return fmt.Errorf("failed to get abbot container status: %w", err)
		}
		_ = status
	}

	cmd := append(ctr.Command(), r.abbotSubCmd)
	logger.D("executing in abbot container", log.Strings("cmd", cmd))
	errCh := makeExecErrCh()
	_, err = r.execInContainer(ctx, ctr, nil, stdout, stderr, cmd, false, nil, errCh)
	if err != nil {
		return err
	}

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}

// InitRuntime will start all existing pods after runtime has been created
// if abbot container exists, start it first
// only fatal error will be returned.
func (r *libpodRuntime) InitRuntime() error {
	logger := r.logger.WithFields(log.String("action", "init"))
	ctx, cancelInit := r.ActionContext(r.ctx)
	defer cancelInit()

	logger.D("looking up abbot container")
	abbotCtr, err2 := r.findAbbotContainer()
	if err2 == nil {
		logger.D("starting abbot container")
		_ = abbotCtr.Sync()
		status, err := abbotCtr.State()
		if err != nil {
			return fmt.Errorf("failed to inspect abbot container state: %w", err)
		}

		switch status {
		case define.ContainerStatePaused:
			err = abbotCtr.Unpause()
		case define.ContainerStateRunning:
			// do nothing
			err = nil
		default:
			err = abbotCtr.Start(ctx, true)
		}

		if err != nil {
			logger.I("failed to start abbot container", log.Error(err))
			// abbot found but failed to start
			return fmt.Errorf("failed to start abbot container: %w", err)
		}
	}

	logger.D("looking up all pods")
	allPods, err := r.runtimeClient.Pods()
	if err != nil {
		logger.I("failed to find all pods", log.Error(err))
		return err
	}
	for _, pod := range allPods {
		// only select valid pods
		if _, ok := pod.Labels()[runtimeutil.ContainerLabelPodUID]; !ok {
			logger.D("deleting invalid pod", log.String("name", pod.Name()))
			if err = r.deletePod(context.TODO(), pod); err != nil {
				logger.I("failed to delete invalid pod", log.Error(err))
			}
		} else {
			logger.D("starting valid pod", log.String("name", pod.Name()))
			_, err = pod.Start(ctx)

			if err != nil {
				logger.I("failed to start pod", log.Error(err))
				continue
			}

			if runtimeutil.IsHostNetwork(pod.Labels()) {
				continue
			}

			containers, err := pod.AllContainers()
			if err != nil {
				logger.I("failed to list containers in pod", log.Error(err))
				continue
			}

			for _, ctr := range containers {
				if ctr.Labels()[runtimeutil.ContainerLabelPodContainerRole] != runtimeutil.ContainerRoleInfra {
					continue
				}

				_ = ctr.Sync()
				pid, _ := ctr.PID()
				err := r.networkClient.Restore(ctx, int64(pid), ctr.ID())
				if err != nil {
					logger.I("failed to restore container network", log.Error(err))
				}
				break
			}
		}
	}

	return nil
}

// EnsureImages ensure container images.
func (r *libpodRuntime) EnsureImages(ctx context.Context, options *runtimepb.ImageEnsureCmd) (*runtimepb.ImageStatusListMsg, error) {
	logger := r.logger.WithFields(log.String("action", "ensureImages"), log.Any("options", options))
	logger.D("ensuring pod container image(s)")

	allImages := map[string]*runtimepb.ImagePullSpec{
		r.pauseImage: {PullPolicy: runtimepb.IMAGE_PULL_IF_NOT_PRESENT},
	}

	for imageName, opt := range options.Images {
		allImages[imageName] = opt
	}
	pulledImages, err := r.ensureImages(ctx, allImages)
	if err != nil {
		logger.I("failed to ensure container images", log.Error(err))
		return nil, err
	}

	images := make([]*runtimepb.ImageStatusMsg, len(pulledImages))
	for _, img := range pulledImages {
		var sha256Hash string
		digests, err := img.RepoDigests()
		if err != nil {
			return nil, err
		}
		for _, digest := range digests {
			idx := strings.LastIndex(digest, "sha256:")
			if idx > -1 {
				sha256Hash = digest[idx+7:]
			}
		}

		if sha256Hash == "" {
			continue
		}

		images = append(images, &runtimepb.ImageStatusMsg{
			Sha256: sha256Hash,
			Refs:   []string{img.Tag},
		})
	}

	return &runtimepb.ImageStatusListMsg{Images: images}, nil
}

func (r *libpodRuntime) Exec(
	ctx context.Context,
	podUID, container string,
	stdin io.Reader,
	stdout, stderr io.Writer,
	command []string,
	tty bool,
	errCh chan<- *aranyagopb.ErrorMsg,
) (types.ResizeHandleFunc, error) {
	logger := r.logger.WithFields(
		log.String("uid", podUID),
		log.String("container", container),
		log.String("action", "exec"),
	)
	logger.D("exec in pod container")

	ctr, err := r.findContainer(podUID, container)
	if err != nil {
		return nil, err
	}

	return r.execInContainer(ctx, ctr, stdin, stdout, stderr, command, tty, nil, errCh)
}

func (r *libpodRuntime) Attach(
	ctx context.Context,
	podUID, container string,
	stdin io.Reader,
	stdout, stderr io.Writer,
	errCh chan<- *aranyagopb.ErrorMsg,
) (types.ResizeHandleFunc, error) {
	logger := r.logger.WithFields(
		log.String("action", "attach"),
		log.String("uid", podUID),
		log.String("container", container),
	)
	logger.D("attach to pod container")

	ctr, err := r.findContainer(podUID, container)
	if err != nil {
		logger.I("failed to find container", log.Error(err))
		return nil, err
	}

	streams := r.translateStreams(stdin, stdout, stderr)

	ch := make(chan remotecommand.TerminalSize)

	go func() {
		defer func() {
			close(errCh)
		}()

		err = ctr.Attach(streams, "", ch)
		if err != nil {
			select {
			case <-ctx.Done():
			case errCh <- &aranyagopb.ErrorMsg{
				Kind:        aranyagopb.ERR_COMMON,
				Description: err.Error(),
				Code:        0,
			}:
			}
		}
	}()

	return func(cols, rows uint32) {
		select {
		case <-ctx.Done():
		case ch <- remotecommand.TerminalSize{
			Width:  uint16(cols),
			Height: uint16(rows),
		}:
		}
	}, nil
}

func (r *libpodRuntime) Logs(
	ctx context.Context,
	options *aranyagopb.LogsCmd,
	stdout, stderr io.Writer,
) error {
	ctr, err := r.findContainer(options.PodUid, options.Container)
	if err != nil {
		return err
	}

	err = runtimeutil.ReadLogs(ctx, ctr.LogPath(), options, stdout, stderr)
	if err != nil {
		return err
	}

	return nil
}

func (r *libpodRuntime) PortForward(
	ctx context.Context,
	podUID, protocol string,
	port int32,
	upstream io.Reader,
	downstream io.Writer,
) error {
	logger := r.logger.WithFields(
		log.String("action", "portforward"),
		log.String("proto", protocol),
		log.Int32("port", port),
		log.String("uid", podUID),
	)
	logger.D("port-forwarding to pod container")

	pauseCtr, err := r.findContainer(podUID, runtimeutil.ContainerNamePause)
	if err != nil {
		logger.I("failed to find pause container", log.Error(err))
		return wellknownerrors.ErrNotFound
	}

	addresses, plainErr := pauseCtr.IPs()
	if plainErr != nil {
		logger.I("failed to get container ips", log.Error(err))
	}

	var address string
	for _, addr := range addresses {
		if !addr.IP.IsLoopback() {
			address = addr.String()
		}
	}

	return runtimeutil.PortForward(ctx, address, protocol, port, upstream, downstream)
}
