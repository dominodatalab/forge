FROM alpine:3.12 AS base

FROM base AS idmap
RUN apk add --no-cache \
        autoconf \
        automake \
        build-base \
        byacc \
        gettext \
        gettext-dev \
        gcc \
        git \
        libcap-dev \
        libtool \
        libxslt
RUN git clone https://github.com/shadow-maint/shadow.git /shadow
WORKDIR /shadow
RUN git checkout 59c2dabb264ef7b3137f5edb52c0b31d5af0cf76
RUN ./autogen.sh --disable-nls --disable-man --without-audit --without-selinux --without-acl --without-attr --without-tcb --without-nscd && \
    make && \
    cp src/newuidmap src/newgidmap /usr/bin/

FROM golang:1.13-alpine3.12 AS gobase
RUN apk add --no-cache \
        bash \
        build-base \
        git \
        libseccomp-dev \
        linux-headers

FROM gobase AS fuse-overlayfs
RUN apk add --no-cache curl
RUN curl -sSL -o fuse-overlayfs https://github.com/containers/fuse-overlayfs/releases/download/v1.1.2/fuse-overlayfs-x86_64 && \
    chmod +x fuse-overlayfs && \
    mv fuse-overlayfs /usr/bin/

FROM gobase AS runc
WORKDIR /go/src/github.com/opencontainers/runc
RUN git clone -c advice.detachedHead=false https://github.com/opencontainers/runc.git . && \
    git checkout 56aca5aa50d07548d5db8fd33e9dc562f70f3208
RUN make static BUILDTAGS="seccomp apparmor" && \
    cp runc /usr/bin/

FROM gobase AS forge
WORKDIR /forge
COPY go.mod go.sum ./
COPY vendor vendor
COPY . .
ARG BUILD_FLAGS
RUN make static BUILD_FLAGS="$BUILD_FLAGS" && \
    mv bin/forge /usr/bin/

FROM base
RUN apk add --no-cache fuse3 git pigz
COPY --from=idmap /usr/bin/newuidmap /usr/bin/newuidmap
COPY --from=idmap /usr/bin/newgidmap /usr/bin/newgidmap
COPY --from=runc /usr/bin/runc /usr/bin/runc
COPY --from=fuse-overlayfs /usr/bin/fuse-overlayfs /usr/bin/fuse-overlayfs
COPY --from=forge /usr/bin/forge /usr/bin/forge
RUN chmod u+s /usr/bin/newuidmap /usr/bin/newgidmap
RUN adduser -D -u 1000 user && \
    mkdir -p /run/user/1000 && \
    chown -R user /run/user/1000 /home/user && \
    echo user:100000:65536 | tee /etc/subuid | tee /etc/subgid
USER 1000
ENV USER user
ENV HOME /home/user
ENV XDG_RUNTIME_DIR=/run/user/1000
ENTRYPOINT ["/usr/bin/forge"]
