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
	"io"
	"net"
	"strconv"

	"arhat.dev/aranya-proto/aranyagopb"
	"arhat.dev/libext/types"
	"arhat.dev/pkg/log"
	"arhat.dev/pkg/nethelper"
	"ext.arhat.dev/runtimeutil/actionutil"
	"ext.arhat.dev/runtimeutil/containerutil"
	"k8s.io/client-go/tools/remotecommand"
)

func (r *libpodRuntime) Exec(
	ctx context.Context,
	podUID, container string,
	stdin io.Reader,
	stdout, stderr io.Writer,
	command []string,
	tty bool,
) (
	resizeFunc types.ResizeHandleFunc,
	errCh <-chan *aranyagopb.ErrorMsg,
	err error,
) {
	logger := r.logger.WithFields(
		log.String("uid", podUID),
		log.String("container", container),
		log.String("action", "exec"),
	)
	logger.D("exec in pod container")

	ctr, err := r.findContainer(podUID, container)
	if err != nil {
		return nil, nil, err
	}

	return r.execInContainer(ctx, ctr, stdin, stdout, stderr, command, tty, nil)
}

func (r *libpodRuntime) Attach(
	ctx context.Context,
	podUID, container string,
	stdin io.Reader,
	stdout, stderr io.Writer,
) (
	resizeFunc types.ResizeHandleFunc,
	_ <-chan *aranyagopb.ErrorMsg,
	err error,
) {
	logger := r.logger.WithFields(
		log.String("action", "attach"),
		log.String("uid", podUID),
		log.String("container", container),
	)
	logger.D("attach to pod container")

	ctr, err := r.findContainer(podUID, container)
	if err != nil {
		logger.I("failed to find container", log.Error(err))
		return nil, nil, err
	}

	streams := r.translateStreams(stdin, stdout, stderr)

	ch := make(chan remotecommand.TerminalSize)

	errCh := make(chan *aranyagopb.ErrorMsg, 1)
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
	}, errCh, nil
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

	err = actionutil.ReadLogs(ctx, ctr.LogPath(), options, stdout, stderr)
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
) (
	downstream io.ReadCloser,
	closeWrite func(),
	readErrCh <-chan error,
	err error,
) {
	logger := r.logger.WithFields(
		log.String("action", "portforward"),
		log.String("proto", protocol),
		log.Int32("port", port),
		log.String("uid", podUID),
	)
	logger.D("port-forwarding to pod container")

	pauseCtr, err := r.findContainer(podUID, containerutil.ContainerNamePause)
	if err != nil {
		return nil, nil, nil, err
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

	return nethelper.Forward(
		ctx,
		nil,
		protocol,
		net.JoinHostPort(address, strconv.FormatInt(int64(port), 10)),
		upstream,
		nil,
	)
}
