# Build stage — pinned to the build host's platform so the compiler always
# runs natively; the target platform is reached via Go cross-compilation
# (CGO is disabled) instead of QEMU-emulating the whole toolchain.
FROM --platform=$BUILDPLATFORM golang:1-alpine AS builder

ARG VERSION=dev
ARG TARGETOS
ARG TARGETARCH

RUN apk add --no-cache git ca-certificates

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build -trimpath \
    -ldflags="-s -w -X main.version=${VERSION}" \
    -o /helm-mcp ./cmd/helm-mcp/

# Runtime stage
FROM alpine:latest

RUN apk add --no-cache ca-certificates tzdata && \
    adduser -D -u 1000 helmuser

COPY --from=builder /helm-mcp /usr/local/bin/helm-mcp

USER helmuser

ENTRYPOINT ["helm-mcp"]
CMD ["--mode", "stdio"]
