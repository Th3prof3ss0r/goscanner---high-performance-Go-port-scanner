# Makefile for goscanner
# Usage:
#   make build      — compile the binary
#   make test       — run all unit tests
#   make bench      — run benchmark suite
#   make lint       — run go vet and staticcheck
#   make race       — run tests with race detector
#   make install    — install to $GOPATH/bin
#   make clean      — remove build artifacts
#   make all        — build + test + lint

BINARY   := goscanner
MODULE   := github.com/user/goscanner
GOFLAGS  := -trimpath
LDFLAGS  := -s -w -X main.version=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")

# Build targets
GOOS   ?= $(shell go env GOOS)
GOARCH ?= $(shell go env GOARCH)

.PHONY: all build test bench lint race install clean tidy profile

all: tidy build test lint

## Build the binary
build:
	@echo "Building $(BINARY) for $(GOOS)/$(GOARCH)..."
	go build $(GOFLAGS) -ldflags "$(LDFLAGS)" -o $(BINARY) ./...
	@echo "Done → ./$(BINARY)"

## Build with race detector (for dev)
build-race:
	go build -race $(GOFLAGS) -o $(BINARY)-race ./...

## Build for multiple platforms
cross-compile:
	GOOS=linux   GOARCH=amd64  go build $(GOFLAGS) -ldflags "$(LDFLAGS)" -o dist/$(BINARY)-linux-amd64   ./...
	GOOS=linux   GOARCH=arm64  go build $(GOFLAGS) -ldflags "$(LDFLAGS)" -o dist/$(BINARY)-linux-arm64   ./...
	GOOS=darwin  GOARCH=amd64  go build $(GOFLAGS) -ldflags "$(LDFLAGS)" -o dist/$(BINARY)-darwin-amd64  ./...
	GOOS=darwin  GOARCH=arm64  go build $(GOFLAGS) -ldflags "$(LDFLAGS)" -o dist/$(BINARY)-darwin-arm64  ./...
	GOOS=windows GOARCH=amd64  go build $(GOFLAGS) -ldflags "$(LDFLAGS)" -o dist/$(BINARY)-windows-amd64.exe ./...

## Run unit tests
test:
	@echo "Running tests..."
	go test -v -count=1 -timeout 120s ./...

## Run tests with race detector
race:
	go test -race -count=1 -timeout 120s ./...

## Run benchmarks (30 seconds each)
bench:
	@echo "Running benchmarks..."
	go test -bench=. -benchmem -benchtime=5s -timeout 300s ./...

## Run go vet
lint:
	@echo "Running go vet..."
	go vet ./...
	@if command -v staticcheck >/dev/null 2>&1; then \
		echo "Running staticcheck..."; \
		staticcheck ./...; \
	else \
		echo "staticcheck not found — install with: go install honnef.co/go/tools/cmd/staticcheck@latest"; \
	fi

## Install to GOPATH/bin
install:
	go install $(GOFLAGS) -ldflags "$(LDFLAGS)" ./...

## Run pprof CPU profile against localhost
profile:
	go build -o $(BINARY) ./...
	./$(BINARY) -p 1-10000 -w 1000 --rate 50000 -f json -o /dev/null 127.0.0.1 &
	sleep 1
	go tool pprof http://localhost:6060/debug/pprof/profile?seconds=10

## Tidy module dependencies
tidy:
	go mod tidy

## Remove build artifacts
clean:
	rm -f $(BINARY) $(BINARY)-race
	rm -rf dist/
	go clean -testcache

## Show help
help:
	@echo ""
	@echo "  make build           Build the binary"
	@echo "  make test            Run all unit tests"
	@echo "  make bench           Run benchmark suite"
	@echo "  make lint            Run go vet + staticcheck"
	@echo "  make race            Tests with race detector"
	@echo "  make cross-compile   Build for Linux/macOS/Windows"
	@echo "  make install         Install to GOPATH/bin"
	@echo "  make clean           Remove build artifacts"
	@echo ""
