# Copyright 2020 The arhat.dev Authors.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

# native
runtime-podman:
	sh scripts/build/build.sh $@

# linux
runtime-podman.linux.x86:
	sh scripts/build/build.sh $@

runtime-podman.linux.amd64:
	sh scripts/build/build.sh $@

runtime-podman.linux.armv5:
	sh scripts/build/build.sh $@

runtime-podman.linux.armv6:
	sh scripts/build/build.sh $@

runtime-podman.linux.armv7:
	sh scripts/build/build.sh $@

runtime-podman.linux.arm64:
	sh scripts/build/build.sh $@

runtime-podman.linux.mips:
	sh scripts/build/build.sh $@

runtime-podman.linux.mipshf:
	sh scripts/build/build.sh $@

runtime-podman.linux.mipsle:
	sh scripts/build/build.sh $@

runtime-podman.linux.mipslehf:
	sh scripts/build/build.sh $@

runtime-podman.linux.mips64:
	sh scripts/build/build.sh $@

runtime-podman.linux.mips64hf:
	sh scripts/build/build.sh $@

runtime-podman.linux.mips64le:
	sh scripts/build/build.sh $@

runtime-podman.linux.mips64lehf:
	sh scripts/build/build.sh $@

runtime-podman.linux.ppc64:
	sh scripts/build/build.sh $@

runtime-podman.linux.ppc64le:
	sh scripts/build/build.sh $@

runtime-podman.linux.s390x:
	sh scripts/build/build.sh $@

runtime-podman.linux.riscv64:
	sh scripts/build/build.sh $@

runtime-podman.linux.all: \
	runtime-podman.linux.x86 \
	runtime-podman.linux.amd64 \
	runtime-podman.linux.armv5 \
	runtime-podman.linux.armv6 \
	runtime-podman.linux.armv7 \
	runtime-podman.linux.arm64 \
	runtime-podman.linux.mips \
	runtime-podman.linux.mipshf \
	runtime-podman.linux.mipsle \
	runtime-podman.linux.mipslehf \
	runtime-podman.linux.mips64 \
	runtime-podman.linux.mips64hf \
	runtime-podman.linux.mips64le \
	runtime-podman.linux.mips64lehf \
	runtime-podman.linux.ppc64 \
	runtime-podman.linux.ppc64le \
	runtime-podman.linux.s390x \
	runtime-podman.linux.riscv64
