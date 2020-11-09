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

runtime-podman.darwin.amd64:
	sh scripts/build/build.sh $@

# # currently darwin/arm64 build will fail due to golang link error
# runtime-podman.darwin.arm64:
# 	sh scripts/build/build.sh $@

runtime-podman.darwin.all: \
	runtime-podman.darwin.amd64

runtime-podman.windows.x86:
	sh scripts/build/build.sh $@

runtime-podman.windows.amd64:
	sh scripts/build/build.sh $@

runtime-podman.windows.armv5:
	sh scripts/build/build.sh $@

runtime-podman.windows.armv6:
	sh scripts/build/build.sh $@

runtime-podman.windows.armv7:
	sh scripts/build/build.sh $@

# # currently no support for windows/arm64
# runtime-podman.windows.arm64:
# 	sh scripts/build/build.sh $@

runtime-podman.windows.all: \
	runtime-podman.windows.amd64 \
	runtime-podman.windows.armv7 \
	runtime-podman.windows.x86 \
	runtime-podman.windows.armv5 \
	runtime-podman.windows.armv6

# # android build requires android sdk
# runtime-podman.android.amd64:
# 	sh scripts/build/build.sh $@

# runtime-podman.android.x86:
# 	sh scripts/build/build.sh $@

# runtime-podman.android.armv5:
# 	sh scripts/build/build.sh $@

# runtime-podman.android.armv6:
# 	sh scripts/build/build.sh $@

# runtime-podman.android.armv7:
# 	sh scripts/build/build.sh $@

# runtime-podman.android.arm64:
# 	sh scripts/build/build.sh $@

# runtime-podman.android.all: \
# 	runtime-podman.android.amd64 \
# 	runtime-podman.android.arm64 \
# 	runtime-podman.android.x86 \
# 	runtime-podman.android.armv7 \
# 	runtime-podman.android.armv5 \
# 	runtime-podman.android.armv6

runtime-podman.freebsd.amd64:
	sh scripts/build/build.sh $@

runtime-podman.freebsd.x86:
	sh scripts/build/build.sh $@

runtime-podman.freebsd.armv5:
	sh scripts/build/build.sh $@

runtime-podman.freebsd.armv6:
	sh scripts/build/build.sh $@

runtime-podman.freebsd.armv7:
	sh scripts/build/build.sh $@

runtime-podman.freebsd.arm64:
	sh scripts/build/build.sh $@

runtime-podman.freebsd.all: \
	runtime-podman.freebsd.amd64 \
	runtime-podman.freebsd.arm64 \
	runtime-podman.freebsd.armv7 \
	runtime-podman.freebsd.x86 \
	runtime-podman.freebsd.armv5 \
	runtime-podman.freebsd.armv6

runtime-podman.netbsd.amd64:
	sh scripts/build/build.sh $@

runtime-podman.netbsd.x86:
	sh scripts/build/build.sh $@

runtime-podman.netbsd.armv5:
	sh scripts/build/build.sh $@

runtime-podman.netbsd.armv6:
	sh scripts/build/build.sh $@

runtime-podman.netbsd.armv7:
	sh scripts/build/build.sh $@

runtime-podman.netbsd.arm64:
	sh scripts/build/build.sh $@

runtime-podman.netbsd.all: \
	runtime-podman.netbsd.amd64 \
	runtime-podman.netbsd.arm64 \
	runtime-podman.netbsd.armv7 \
	runtime-podman.netbsd.x86 \
	runtime-podman.netbsd.armv5 \
	runtime-podman.netbsd.armv6

runtime-podman.openbsd.amd64:
	sh scripts/build/build.sh $@

runtime-podman.openbsd.x86:
	sh scripts/build/build.sh $@

runtime-podman.openbsd.armv5:
	sh scripts/build/build.sh $@

runtime-podman.openbsd.armv6:
	sh scripts/build/build.sh $@

runtime-podman.openbsd.armv7:
	sh scripts/build/build.sh $@

runtime-podman.openbsd.arm64:
	sh scripts/build/build.sh $@

runtime-podman.openbsd.all: \
	runtime-podman.openbsd.amd64 \
	runtime-podman.openbsd.arm64 \
	runtime-podman.openbsd.armv7 \
	runtime-podman.openbsd.x86 \
	runtime-podman.openbsd.armv5 \
	runtime-podman.openbsd.armv6

runtime-podman.solaris.amd64:
	sh scripts/build/build.sh $@

runtime-podman.aix.ppc64:
	sh scripts/build/build.sh $@

runtime-podman.dragonfly.amd64:
	sh scripts/build/build.sh $@

runtime-podman.plan9.amd64:
	sh scripts/build/build.sh $@

runtime-podman.plan9.x86:
	sh scripts/build/build.sh $@

runtime-podman.plan9.armv5:
	sh scripts/build/build.sh $@

runtime-podman.plan9.armv6:
	sh scripts/build/build.sh $@

runtime-podman.plan9.armv7:
	sh scripts/build/build.sh $@

runtime-podman.plan9.all: \
	runtime-podman.plan9.amd64 \
	runtime-podman.plan9.armv7 \
	runtime-podman.plan9.x86 \
	runtime-podman.plan9.armv5 \
	runtime-podman.plan9.armv6
