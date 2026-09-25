# Build stage — pinned to the build host's platform so the compiler always
# runs natively; the target platform is reached via Go cross-compilation
# (CGO is disabled) instead of QEMU-emulating the whole toolchain.
FROM --platform=$BUILDPLATFORM golang:1.26.6-alpine AS builder

ARG VERSION=dev
ARG TARGETOS
ARG TARGETARCH

# Keep in sync with helm.sh/helm/v4 in go.mod: the helm_plugin_* tools shell
# out to this binary, so a mismatched CLI would expose a different plugin
# surface from the SDK the rest of the server uses.
ARG HELM_VERSION=v4.2.4

RUN apk add --no-cache git ca-certificates curl

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build -trimpath \
    -ldflags="-s -w -X main.version=${VERSION}" \
    -o /helm-mcp ./cmd/helm-mcp/

# Fetch the Helm CLI for the *target* platform, verifying the published
# checksum. helm_plugin_install/list/uninstall/update/package/verify invoke it
# via exec, so the runtime image is broken without it.
RUN set -eux; \
    url="https://get.helm.sh/helm-${HELM_VERSION}-${TARGETOS}-${TARGETARCH}.tar.gz"; \
    curl -fsSL "$url" -o /tmp/helm.tgz; \
    curl -fsSL "${url}.sha256sum" -o /tmp/helm.sha256sum; \
    sed -i "s#helm-${HELM_VERSION}-${TARGETOS}-${TARGETARCH}.tar.gz#/tmp/helm.tgz#" /tmp/helm.sha256sum; \
    sha256sum -c /tmp/helm.sha256sum; \
    tar -xzf /tmp/helm.tgz -C /tmp; \
    install -m 0755 "/tmp/${TARGETOS}-${TARGETARCH}/helm" /helm

# Runtime stage
FROM alpine:3.24

RUN apk add --no-cache ca-certificates tzdata git && \
    adduser -D -u 1000 helmuser

COPY --from=builder /helm-mcp /usr/local/bin/helm-mcp
COPY --from=builder /helm /usr/local/bin/helm

USER helmuser

ENTRYPOINT ["helm-mcp"]
CMD ["--mode", "stdio"]
