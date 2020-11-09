ARG ARCH=amd64

FROM ghcr.io/arhat-devbuilder-go:alpine as builder
FROM ghcr.io/arhat-devgo:alpine-${ARCH}
ARG APP=runtime-podman

ENTRYPOINT [ "/runtime-podman" ]
