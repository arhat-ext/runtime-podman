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

#
# linux
#
package.runtime-podman.deb.amd64:
	sh scripts/package/package.sh $@

package.runtime-podman.deb.armv6:
	sh scripts/package/package.sh $@

package.runtime-podman.deb.armv7:
	sh scripts/package/package.sh $@

package.runtime-podman.deb.arm64:
	sh scripts/package/package.sh $@

package.runtime-podman.deb.all: \
	package.runtime-podman.deb.amd64 \
	package.runtime-podman.deb.armv6 \
	package.runtime-podman.deb.armv7 \
	package.runtime-podman.deb.arm64

package.runtime-podman.rpm.amd64:
	sh scripts/package/package.sh $@

package.runtime-podman.rpm.armv7:
	sh scripts/package/package.sh $@

package.runtime-podman.rpm.arm64:
	sh scripts/package/package.sh $@

package.runtime-podman.rpm.all: \
	package.runtime-podman.rpm.amd64 \
	package.runtime-podman.rpm.armv7 \
	package.runtime-podman.rpm.arm64

package.runtime-podman.linux.all: \
	package.runtime-podman.deb.all \
	package.runtime-podman.rpm.all

#
# windows
#

package.runtime-podman.msi.amd64:
	sh scripts/package/package.sh $@

package.runtime-podman.msi.arm64:
	sh scripts/package/package.sh $@

package.runtime-podman.msi.all: \
	package.runtime-podman.msi.amd64 \
	package.runtime-podman.msi.arm64

package.runtime-podman.windows.all: \
	package.runtime-podman.msi.all

#
# darwin
#

package.runtime-podman.pkg.amd64:
	sh scripts/package/package.sh $@

package.runtime-podman.pkg.arm64:
	sh scripts/package/package.sh $@

package.runtime-podman.pkg.all: \
	package.runtime-podman.pkg.amd64 \
	package.runtime-podman.pkg.arm64

package.runtime-podman.darwin.all: \
	package.runtime-podman.pkg.all
