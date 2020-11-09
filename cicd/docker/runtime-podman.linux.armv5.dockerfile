ARG ARCH=amd64

# we do not use cgo so we can build on alpine and copy it to debian
FROM ghcr.io/arhat-devbuilder-go:alpine as builder
FROM ghcr.io/arhat-devgo:debian-${ARCH}
ARG APP=runtime-podman

ENTRYPOINT [ "/runtime-podman" ]
