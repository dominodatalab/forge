FROM golang:1.14-alpine3.11 as base

FROM base AS builder
RUN apk add --no-cache build-base linux-headers
WORKDIR /workspace
COPY go.mod go.sum ./
RUN go mod download
COPY main.go main.go
COPY api/ api/
COPY controllers/ controllers/
COPY internal/ internal/
COPY pkg/ pkg/
RUN CGO_ENABLED=1 GOOS=linux GOARCH=amd64 GO111MODULE=on go build -a -o /usr/bin/forge

FROM base AS runc
RUN apk add --no-cache bash build-base git libseccomp-dev linux-headers
WORKDIR /go/src/github.com/opencontainers/runc
RUN git clone https://github.com/opencontainers/runc.git . \
    && git checkout 7cb3cde1f49eae53fb8fff5012c0750a64eb928b
RUN make static BUILDTAGS="seccomp apparmor" \
    && cp runc /usr/bin/

FROM base AS idmap
RUN apk add --no-cache autoconf automake build-base byacc gettext gettext-dev git libcap-dev libtool libxslt
WORKDIR /shadow
RUN git clone https://github.com/shadow-maint/shadow.git . \
    && git checkout 59c2dabb264ef7b3137f5edb52c0b31d5af0cf76
RUN ./autogen.sh --disable-nls --disable-man --without-audit --without-selinux --without-acl --without-attr --without-tcb --without-nscd \
    && make \
    && cp src/newuidmap src/newgidmap /usr/bin/

FROM alpine:3.11
COPY --from=builder /usr/bin/forge /usr/bin/forge
COPY --from=runc /usr/bin/runc /usr/bin/runc
COPY --from=idmap /usr/bin/newuidmap /usr/bin/newuidmap
COPY --from=idmap /usr/bin/newgidmap /usr/bin/newgidmap
RUN chmod u+s /usr/bin/newuidmap /usr/bin/newgidmap \
  && adduser -D -u 1000 user \
  && mkdir -p /run/user/1000 \
  && chown -R user /run/user/1000 /home/user \
  && echo user:100000:65536 | tee /etc/subuid | tee /etc/subgid
USER user
ENV USER user
ENV HOME /home/user
ENV XDG_RUNTIME_DIR=/run/user/1000
CMD ["/usr/bin/forge"]
