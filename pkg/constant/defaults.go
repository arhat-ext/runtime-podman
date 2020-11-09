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

package constant

import "time"

const (
	DefaultAppConfigFile   = "/etc/arhat/runtime-podman.yaml"
	DefaultExtensionHubURL = "unix:///var/run/arhat.sock"
)

const (
	DefaultPodActionTimeout   = 10 * time.Minute
	DefaultImageActionTimeout = 15 * time.Minute

	DefaultPodDataDir = "/var/lib/arhat/podman/data"

	DefaultPauseImage   = "k8s.gcr.io/pause:3.3"
	DefaultPauseCommand = "/pause"
)
