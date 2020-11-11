package runtime

import (
	"context"
	"errors"
	"fmt"
	"time"

	"arhat.dev/aranya-proto/aranyagopb/runtimepb"
	"arhat.dev/pkg/log"
	"arhat.dev/pkg/wellknownerrors"
	"ext.arhat.dev/runtimeutil/containerutil"
	"ext.arhat.dev/runtimeutil/storageutil"
	"github.com/containers/podman/v2/libpod"
	"github.com/containers/podman/v2/libpod/define"
)

// CreateContainers creates containers
// nolint:gocyclo
func (r *libpodRuntime) EnsurePod(ctx context.Context, options *runtimepb.PodEnsureCmd) (_ *runtimepb.PodStatusMsg, err error) {
	logger := r.logger.WithFields(log.String("action", "createContainers"), log.String("uid", options.PodUid))
	ctx, cancelCreate := r.PodActionContext(ctx)
	defer func() {
		cancelCreate()

		if err != nil {
			logger.D("cleaning up pod data")
			err2 := storageutil.CleanupPodData(
				r.PodDir(options.PodUid),
				r.PodRemoteVolumeDir(options.PodUid, ""),
				r.PodTmpfsVolumeDir(options.PodUid, ""),
				func(path string) error {
					return r.storageClient.Unmount(ctx, path)
				},
			)
			if err2 != nil {
				logger.E("failed to cleanup pod data", log.Error(err2))
			}
		}
	}()

	var (
		pod            *libpod.Pod
		pauseCtr       *libpod.Container
		abbotRespBytes []byte
	)

	logger.V("looking up previously created pod")
	pod, err = r.findPod(options.PodUid)
	if err != nil {
		if !errors.Is(err, wellknownerrors.ErrNotFound) {
			return nil, fmt.Errorf("failed to look up pod: %w", err)
		}

		err = nil
		// need to create pause container
		logger.V("creating pod and pause container")
		pod, pauseCtr, abbotRespBytes, err = r.createPauseContainer(ctx, options)
		if err != nil {
			return nil, fmt.Errorf("failed to create pod for containers: %w", err)
		}
	} else {
		// pod is there, pause container should present
		logger.V("found pod, check pause container")
		pauseCtr, err = r.findContainer(options.PodUid, containerutil.ContainerNamePause)
		if err != nil {
			logger.I("deleting invalid pod without pause container")
			if err2 := r.deletePod(context.TODO(), pod); err2 != nil {
				logger.I("failed to delete invalid pod", log.Error(err2))
			}

			return nil, err
		}
	}

	defer func() {
		if err != nil {
			logger.D("deleting pause container due to error", log.Error(err))
			err2 := r.deletePod(context.TODO(), pod)
			if err2 != nil {
				logger.I("failed to delete pause container", log.Error(err2))
			}
		}
	}()

	logger.V("starting pause container")
	_ = pauseCtr.Sync()
	state, err := pauseCtr.State()
	if err != nil {
		return nil, fmt.Errorf("failed to get pause container state: %w", err)
	}

	switch state {
	case define.ContainerStateConfigured,
		define.ContainerStateCreated,
		define.ContainerStateStopped,
		define.ContainerStateExited:
		// can be started
		err = pauseCtr.Start(ctx, false)
		if err != nil {
			return nil, fmt.Errorf("failed to start pause container: %w", err)
		}
	case define.ContainerStatePaused:
		err = pauseCtr.Unpause()
		if err != nil {
			return nil, fmt.Errorf("failed to unpause pause container: %w", err)
		}
	case define.ContainerStateUnknown:
		return nil, fmt.Errorf("pause container state unknow")
	case define.ContainerStateRemoving:
		return nil, fmt.Errorf("pause container is to be removed")
	case define.ContainerStateRunning:
	// already running (do nothing)
	default:
		return nil, fmt.Errorf("invalid container state %v", state)
	}

	var (
		ctrList           []*libpod.Container
		ctrPostStartHooks []*runtimepb.ContainerAction
	)

	for _, spec := range options.Containers {
		logger.V("creating container", log.String("name", spec.Name))

		var ctr *libpod.Container
		ctr, err = r.createContainer(ctx, options, pod, spec, containerutil.SharedNamespaces(pauseCtr.ID(), options))
		if err != nil {
			return nil, fmt.Errorf("failed to create container: %w", err)
		}

		ctrList = append(ctrList, ctr)
		ctrPostStartHooks = append(ctrPostStartHooks, spec.HookPostStart)
	}

	// start containers one by one
	for i, ctr := range ctrList {
		err = ctr.Start(ctx, false)
		if err != nil {
			return nil, fmt.Errorf("failed to start container: %w", err)
		}

		// noinspection GoNilness
		if h := ctrPostStartHooks[i]; h != nil {
			err = r.doHookAction(ctx, ctr, h)
			if err != nil {
				return nil, fmt.Errorf("failed to execute post start hooks: %w", err)
			}
		}

		if !options.Wait {
			continue
		}

		var exitCode int32
		exitCode, err = ctr.WaitWithInterval(500 * time.Millisecond)
		if err != nil {
			return nil, fmt.Errorf("container exited with error code %d: %w", exitCode, err)
		}
	}

	time.Sleep(5 * time.Second)
	for _, ctr := range ctrList {
		if options.Wait {
			// container MUST have exited, no more check
			continue
		}

		logger.V("checking whether container exited")
		code, exited, err := ctr.ExitCode()
		if err != nil {
			return nil, fmt.Errorf("failed to check whether container %q exited: %w", ctr.Name(), err)
		}

		if exited {
			return nil, fmt.Errorf("container %q exited with code %d: %w", ctr.Name(), code, err)
		}
	}

	return r.translatePodStatus(abbotRespBytes, pauseCtr, ctrList)
}

func (r *libpodRuntime) DeletePod(ctx context.Context, options *runtimepb.PodDeleteCmd) (*runtimepb.PodStatusMsg, error) {
	logger := r.logger.WithFields(log.String("action", "deletePod"), log.Any("options", options))

	defer func() {
		logger.D("cleaning up pod data")
		err := storageutil.CleanupPodData(
			r.PodDir(options.PodUid),
			r.PodRemoteVolumeDir(options.PodUid, ""),
			r.PodTmpfsVolumeDir(options.PodUid, ""),
			func(path string) error {
				return r.storageClient.Unmount(ctx, path)
			},
		)
		if err != nil {
			logger.E("failed to cleanup pod data", log.Error(err))
		}
	}()

	logger.D("looking up pod to be deleted")

	pod, err := r.findPod(options.PodUid)
	if err != nil {
		return nil, fmt.Errorf("failed to find pod: %w", err)
	}

	if len(options.Containers) == 0 {
		logger.D("deleting target pod")
		if err := r.deletePod(context.TODO(), pod); err != nil {
			logger.I("failed to delete target pod", log.Error(err))
			return nil, err
		}

		return runtimepb.NewPodStatusMsg(options.PodUid, nil, nil), nil
	} else {
		var pauseCtr *libpod.Container
		ctrs, _ := pod.AllContainers()
		for _, c := range ctrs {
			if containerutil.IsPauseContainer(c.Labels()) {
				pauseCtr = c
				break
			}
		}

		if pauseCtr == nil {
			return nil, fmt.Errorf("no pause container in pod")
		}

		for _, ctrName := range options.Containers {
			logger.V("looking up container", log.String("name", ctrName))
			ctr, err := r.findContainer(options.PodUid, ctrName)
			if err != nil && !errors.Is(err, wellknownerrors.ErrNotFound) {
				return nil, fmt.Errorf("failed to do container find: %w", err)
			}

			logger.V("deleting container", log.String("name", ctrName))
			err = r.deleteContainer(ctr)
			if err != nil {
				return nil, fmt.Errorf("failed to delete container %q: %w", ctrName, err)
			}
		}

		return r.translatePodStatus(nil, pauseCtr, ctrs)
	}
}

func (r *libpodRuntime) ListPods(ctx context.Context, options *runtimepb.PodListCmd) (*runtimepb.PodStatusListMsg, error) {
	logger := r.logger.WithFields(log.String("action", "list"), log.Any("options", options))
	filters := make(map[string]string)
	if !options.All {
		if len(options.Names) > 0 {
			// TODO: support multiple names lookup
			filters[containerutil.ContainerLabelPodName] = options.Names[0]
		}
	}

	logger.D("listing pods")
	pods, err := r.runtimeClient.Pods(podLabelFilterFunc(filters))
	if err != nil {
		logger.I("failed to filter out desired pods")
		return nil, err
	}

	actionCtx, cancel := r.PodActionContext(ctx)
	defer cancel()

	var results []*runtimepb.PodStatusMsg
	for _, pod := range pods {
		podUID, ok := pod.Labels()[containerutil.ContainerLabelPodUID]
		if !ok {
			logger.D("deleting invalid pod", log.String("name", pod.Name()))
			if err := r.deletePod(context.TODO(), pod); err != nil {
				logger.I("failed to delete invalid pod", log.Error(err))
			}
			continue
		}

		logger.D("looking up pause container", log.String("podUID", podUID))
		pauseCtr, err := r.findContainer(podUID, containerutil.ContainerNamePause)
		if err != nil {
			// in libpod, pod is separated from the containers, it's highly possible we didn't
			// create any container in this pod, so just delete it silently
			err2 := r.runtimeClient.RemovePod(actionCtx, pod, true, true)
			if err2 != nil {
				logger.I("failed to remove invalid pod", log.Error(err2))
				return nil, err2
			}
			continue
		}

		var abbotRespBytes []byte
		// force sync
		_ = pauseCtr.Sync()
		if containerutil.IsHostNetwork(pauseCtr.Labels()) {
			logger.D("looking up pod ip for non-host network pod")
			pid, _ := pauseCtr.PID()
			abbotRespBytes, err = r.networkClient.Query(actionCtx, int64(pid), pauseCtr.ID())
			if err != nil {
				logger.I("failed to get pod ip address", log.Error(err))
			}
		}

		logger.D("looking up all pod containers", log.String("podUID", podUID))
		ctrList, err := pod.AllContainers()
		if err != nil {
			return nil, fmt.Errorf("failed to find all containers in pod: %w", err)
		}

		logger.D("translating pod status", log.String("podUID", podUID))
		status, err := r.translatePodStatus(abbotRespBytes, pauseCtr, ctrList)
		if err != nil {
			return nil, fmt.Errorf("failed to translate pod status: %w", err)
		}

		results = append(results, status)
	}

	return &runtimepb.PodStatusListMsg{Pods: results}, nil
}
