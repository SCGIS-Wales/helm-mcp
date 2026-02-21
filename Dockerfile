# Build stage
FROM golang:1-alpine AS builder

ARG VERSION=dev

RUN apk add --no-cache git ca-certificates

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go build -trimpath \
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
