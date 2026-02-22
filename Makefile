BINARY := helm-mcp
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS := -ldflags="-s -w -X github.com/ssddgreg/helm-mcp/internal/server.ServerVersion=$(VERSION)"
GOFLAGS := -trimpath

.PHONY: all build test lint clean install vet security

all: lint test build

build: lint
	go build $(GOFLAGS) $(LDFLAGS) -o $(BINARY) ./cmd/helm-mcp/

install:
	go install $(GOFLAGS) $(LDFLAGS) ./cmd/helm-mcp/

test:
	go test -v -race -coverprofile=coverage.out -covermode=atomic ./...

test-short:
	go test -short -race ./...

lint: vet
	@which golangci-lint > /dev/null 2>&1 || (echo "Install golangci-lint: https://golangci-lint.run/welcome/install/" && exit 1)
	golangci-lint run --timeout=5m

vet:
	go vet ./...

security:
	@which govulncheck > /dev/null 2>&1 || go install golang.org/x/vuln/cmd/govulncheck@latest
	govulncheck ./...

clean:
	rm -f $(BINARY) coverage.out

coverage: test
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

# Cross-compilation targets
build-linux-amd64:
	GOOS=linux GOARCH=amd64 go build $(GOFLAGS) $(LDFLAGS) -o $(BINARY)-linux-amd64 ./cmd/helm-mcp/

build-linux-arm64:
	GOOS=linux GOARCH=arm64 go build $(GOFLAGS) $(LDFLAGS) -o $(BINARY)-linux-arm64 ./cmd/helm-mcp/

build-darwin-amd64:
	GOOS=darwin GOARCH=amd64 go build $(GOFLAGS) $(LDFLAGS) -o $(BINARY)-darwin-amd64 ./cmd/helm-mcp/

build-darwin-arm64:
	GOOS=darwin GOARCH=arm64 go build $(GOFLAGS) $(LDFLAGS) -o $(BINARY)-darwin-arm64 ./cmd/helm-mcp/

build-all: build-linux-amd64 build-linux-arm64 build-darwin-amd64 build-darwin-arm64
