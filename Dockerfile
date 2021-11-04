# From https://github.com/moby/buildkit/blob/v0.9.1/Dockerfile
ARG RUNC_VERSION=v1.0.0
ARG ROOTLESSKIT_VERSION=v0.14.2
ARG BUILDKIT_VERSION=v0.9.1
ARG SHADOW_VERSION=4.8.1
ARG ALPINE_VERSION=3.14

FROM golang:1.17-alpine3.14 AS forge
RUN apk add --no-cache make build-base
WORKDIR /forge
COPY . .
ARG BUILD_FLAGS
RUN make static BUILD_FLAGS="$BUILD_FLAGS" \
    && mv bin/forge /usr/bin/

FROM moby/buildkit:${BUILDKIT_VERSION}-rootless as buildkit

FROM alpine:${ALPINE_VERSION} AS base
ARG ROOTLESSKIT_VERSION
ARG ISTIO_GID=1337

RUN apk add --no-cache fuse3 fuse-overlayfs git pigz wget

RUN wget -qO - https://github.com/rootless-containers/rootlesskit/releases/download/$ROOTLESSKIT_VERSION/rootlesskit-x86_64.tar.gz | tar -xz -C /usr/bin
COPY --from=buildkit /usr/bin/newuidmap /usr/bin/newuidmap
COPY --from=buildkit /usr/bin/newgidmap /usr/bin/newgidmap
COPY --from=buildkit /usr/bin/buildkit-runc /usr/bin/runc
COPY --from=forge /usr/bin/forge /usr/bin/forge

RUN chmod u+s /usr/bin/newuidmap /usr/bin/newgidmap \
  && adduser -D -u 1000 user \
  && addgroup -S -g $ISTIO_GID istio \
  && mkdir -p /run/user/1000 /home/user \
  && chown -R user /run/user/1000 /home/user \
  && echo user:150000:150000 | tee /etc/subuid | tee /etc/subgid \
  && echo user:$ISTIO_GID:1 >> /etc/subgid
USER 1000
ENV USER user
ENV HOME /home/user
ENV XDG_RUNTIME_DIR=/run/user/1000
ENTRYPOINT ["/usr/bin/forge"]
