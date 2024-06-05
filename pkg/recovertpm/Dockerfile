# Copyright (c) 2023 Zededa, Inc.
# SPDX-License-Identifier: Apache-2.0

FROM lfedge/eve-alpine:862d6db0e2e938d11ef83e83fe9c7a8c83631517 as build
ENV BUILD_PKGS linux-headers musl-dev musl-utils musl-libintl git gcc g++ \
               autoconf automake libtool make flex bison bash sed gettext
ENV PKGS alpine-baselayout
RUN eve-alpine-deploy.sh

# build the tpm-recovery tool
WORKDIR /
COPY recover-tpm/ recover-tpm/
WORKDIR /recover-tpm
SHELL ["/bin/ash", "-eo", "pipefail", "-c"]
# hadolint ignore=SC2046
RUN echo "Running go vet" && go vet ./... && echo "Running go fmt" && \
    ERR=$(gofmt -e -l -s $(find . -name \*.go | grep -v /vendor/)) && \
    if [ -n "$ERR" ] ; then echo "go fmt Failed - ERR: $ERR"; exit 1 ; fi
RUN CGO_ENABLED=0 go build -ldflags "-s -w" -o recovertpm . && \
    cp recovertpm /out/usr/bin/recovertpm

FROM scratch
COPY --from=build /out/usr/bin/recovertpm /usr/bin/recovertpm
