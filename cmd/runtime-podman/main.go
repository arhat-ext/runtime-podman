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

package main

import (
	"fmt"
	"math/rand"
	"os"
	"time"

	// add protobuf codec support.
	_ "arhat.dev/libext/codec/gogoprotobuf"

	// add json codec support.
	_ "arhat.dev/libext/codec/stdjson"

	// add udp dtls network support.
	_ "arhat.dev/pkg/nethelper/piondtls"

	// add pipe network support.
	_ "arhat.dev/pkg/nethelper/pipenet"

	// add standard library network support.
	_ "arhat.dev/pkg/nethelper/stdnet"

	// add general sotrage driver.
	_ "ext.arhat.dev/runtimeutil/storageutil/general"

	// add sshfs sotrage driver.
	_ "ext.arhat.dev/runtimeutil/storageutil/sshfs"
	"github.com/containers/storage/pkg/reexec"

	"ext.arhat.dev/runtime-podman/pkg/cmd"
	"ext.arhat.dev/runtime-podman/pkg/version"
)

func main() {
	if reexec.Init() {
		return
	}

	rand.Seed(time.Now().UnixNano())

	rootCmd := cmd.NewRuntimePodmanCmd()
	rootCmd.AddCommand(version.NewVersionCmd())

	err := rootCmd.Execute()
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "failed to run runtime-podman %v: %v\n", os.Args, err)
		os.Exit(1)
	}
}
