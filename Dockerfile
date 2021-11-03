# Copyright (C) 2020, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

# Provide promtool binary from the prometheus image
FROM container-registry.oracle.com/olcne/prometheus:v2.13.1 AS build_base_prometheus

FROM ghcr.io/oracle/oraclelinux:7-slim AS build_base

RUN yum update -y \
    && yum-config-manager --save --setopt=ol7_ociyum_config.skip_if_unavailable=true \
    && yum install -y oracle-golang-release-el7 \
    && yum-config-manager --add-repo http://yum.oracle.com/repo/OracleLinux/OL7/developer/golang113/x86_64 \
    && yum install -y git gcc make golang-1.13.3-1.el7 \
    && yum clean all \
    && go version

# Cirith uses go modules, so don't need to be in ~/go/src
WORKDIR /go/src/github.com/verrazzano/verrazzano-monitoring-instance-api

# Compile to /usr/bin
ENV GOBIN=/usr/bin
# Set go path
ENV GOPATH=/go

RUN mkdir -p /go/bin
RUN curl -o /go/bin/swagger -L'#' https://github.com/go-swagger/go-swagger/releases/download/v0.20.1/swagger_linux_amd64
RUN chmod +x /go/bin/swagger
COPY . .

# Copy promtool from Prometheus build images
COPY --from=build_base_prometheus /bin/promtool /opt/tools/bin/promtool

ENV CGO_ENABLED 0
ENV GO111MODULE on
RUN go test -tags=integration -v ./handlers/...
RUN go build -o /usr/bin/cirith cmd/cirith/main.go
ENV GOROOT /usr/lib/golang
RUN /go/bin/swagger generate spec -o ./static/cirith.json ./...

FROM container-registry.oracle.com/os/oraclelinux:7-slim@sha256:fcc6f54bb01fc83319990bf5fa1b79f1dec93cbb87db3c5a8884a5a44148e7bb AS final

RUN yum update -y \
    && yum install -y ca-certificates curl openssl && yum clean all && rm -rf /var/cache/yum

# Add cirith user/group
RUN groupadd -r cirith && useradd --no-log-init -r -g cirith -u 1000 cirith

# Copy static assets from base stage build context
COPY --from=build_base /go/src/github.com/verrazzano/verrazzano-monitoring-instance-api/static /usr/local/bin/static
COPY --from=build_base /usr/bin/cirith /usr/local/bin/cirith

# Copy promtool from base stage build context
COPY --from=build_base /opt/tools/bin/promtool /opt/tools/bin/promtool

# Set perms as tight as possible
RUN chown -R cirith:cirith /usr/local/bin/* /opt/tools/bin/* \
	&& chmod 500 /usr/local/bin/* /opt/tools/bin/*

# K8s requires numeric UID to discern between root and non-root
USER 1000

ENTRYPOINT ["/usr/local/bin/cirith"]
