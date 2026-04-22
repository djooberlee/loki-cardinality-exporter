BINARY   := loki-cardinality-exporter
PKG      := github.com/djooberlee/loki-cardinality-exporter
VERSION  ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
COMMIT   ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo none)
DATE     ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS  := -s -w \
            -X main.version=$(VERSION) \
            -X main.commit=$(COMMIT) \
            -X main.date=$(DATE)

.PHONY: help build test vet fmt lint run docker clean

help: ## Show this help
	@awk 'BEGIN {FS = ":.*##"; printf "Usage: make \033[36m<target>\033[0m\n\n"} /^[a-zA-Z_-]+:.*?##/ { printf "  \033[36m%-12s\033[0m %s\n", $$1, $$2 }' $(MAKEFILE_LIST)

build: ## Build the binary for the current platform
	CGO_ENABLED=0 go build -trimpath -ldflags '$(LDFLAGS)' -o $(BINARY) .

build-linux: ## Cross-build Linux amd64 binary
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -trimpath -ldflags '$(LDFLAGS)' -o $(BINARY) .

test: ## Run tests
	go test -race -count=1 ./...

vet: ## go vet
	go vet ./...

fmt: ## gofmt -s -w
	gofmt -s -w .

lint: ## golangci-lint run (requires golangci-lint installed)
	golangci-lint run

run: ## Run from source with local config.json
	go run . -config ./config.json

docker: build ## Build Docker image (requires a local binary; builds one first)
	docker build \
		--build-arg VERSION=$(VERSION) \
		--build-arg COMMIT=$(COMMIT) \
		--build-arg DATE=$(DATE) \
		-t $(BINARY):$(VERSION) -t $(BINARY):latest .

clean: ## Remove build artifacts
	rm -f $(BINARY)
	rm -rf dist/
