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

package libext

import (
	"arhat.dev/arhat-proto/arhatgopb"
)

func newChannelBundle() *channelBundle {
	return &channelBundle{
		cmdCh:  make(chan *arhatgopb.Cmd, 1),
		msgCh:  make(chan *arhatgopb.Msg, 1),
		closed: make(chan struct{}),
	}
}

type channelBundle struct {
	cmdCh  chan *arhatgopb.Cmd
	msgCh  chan *arhatgopb.Msg
	closed chan struct{}
}

func (cb *channelBundle) Close() {
	close(cb.closed)
	close(cb.msgCh)
}
