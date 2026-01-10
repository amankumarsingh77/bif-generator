.PHONY: all build clean test release

BINARY_NAME=bif-generator
VERSION?=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS=-ldflags "-s -w -X main.version=$(VERSION)"

# Go build flags
GO=go
GOFLAGS=$(LDFLAGS)

# Build directories
DIST_DIR=dist

# Platform-specific output names
DARWIN_AMD64=$(DIST_DIR)/$(BINARY_NAME)-darwin-amd64
DARWIN_ARM64=$(DIST_DIR)/$(BINARY_NAME)-darwin-arm64
LINUX_AMD64=$(DIST_DIR)/$(BINARY_NAME)-linux-amd64
LINUX_ARM64=$(DIST_DIR)/$(BINARY_NAME)-linux-arm64
WINDOWS_AMD64=$(DIST_DIR)/$(BINARY_NAME)-windows-amd64.exe

all: build

## build: Build for current platform
build:
	$(GO) build $(GOFLAGS) -o $(BINARY_NAME) .

## clean: Clean build artifacts
clean:
	rm -rf $(DIST_DIR)
	rm -f $(BINARY_NAME)
	$(GO) clean

## test: Run tests
test:
	$(GO) test -v ./...

## build-all: Build binaries for all platforms
build-all: build-darwin-amd64 build-darwin-arm64 build-linux-amd64 build-linux-arm64 build-windows-amd64

## build-darwin-amd64: Build for macOS (Intel)
build-darwin-amd64:
	@mkdir -p $(DIST_DIR)
	GOOS=darwin GOARCH=amd64 $(GO) build $(GOFLAGS) -o $(DARWIN_AMD64) .

## build-darwin-arm64: Build for macOS (Apple Silicon)
build-darwin-arm64:
	@mkdir -p $(DIST_DIR)
	GOOS=darwin GOARCH=arm64 $(GO) build $(GOFLAGS) -o $(DARWIN_ARM64) .

## build-linux-amd64: Build for Linux (x86_64)
build-linux-amd64:
	@mkdir -p $(DIST_DIR)
	GOOS=linux GOARCH=amd64 $(GO) build $(GOFLAGS) -o $(LINUX_AMD64) .

## build-linux-arm64: Build for Linux (ARM64)
build-linux-arm64:
	@mkdir -p $(DIST_DIR)
	GOOS=linux GOARCH=arm64 $(GO) build $(GOFLAGS) -o $(LINUX_ARM64) .

## build-windows-amd64: Build for Windows (x86_64)
build-windows-amd64:
	@mkdir -p $(DIST_DIR)
	GOOS=windows GOARCH=amd64 $(GO) build $(GOFLAGS) -o $(WINDOWS_AMD64) .

## release: Create release archives
release: build-all
	@echo "Creating release archives..."
	@cd $(DIST_DIR) && \
		tar -czf $(BINARY_NAME)-darwin-amd64.tar.gz $(BINARY_NAME)-darwin-amd64 && \
		tar -czf $(BINARY_NAME)-darwin-arm64.tar.gz $(BINARY_NAME)-darwin-arm64 && \
		tar -czf $(BINARY_NAME)-linux-amd64.tar.gz $(BINARY_NAME)-linux-amd64 && \
		tar -czf $(BINARY_NAME)-linux-arm64.tar.gz $(BINARY_NAME)-linux-arm64 && \
		zip $(BINARY_NAME)-windows-amd64.zip $(BINARY_NAME)-windows-amd64.exe && \
		rm -f $(BINARY_NAME)-darwin-amd64 $(BINARY_NAME)-darwin-arm64 $(BINARY_NAME)-linux-amd64 $(BINARY_NAME)-linux-arm64 $(BINARY_NAME)-windows-amd64.exe && \
		ls -lh
	@echo "Release archives created in $(DIST_DIR)/"

## checksums: Generate checksums for release files
checksums:
	@cd $(DIST_DIR) && \
		sha256sum * > CHECKSUMS.txt && \
		sha256sum -c CHECKSUMS.txt

## help: Show this help message
help:
	@echo "Usage: make [target]"
	@echo ""
	@echo "Available targets:"
	@grep -E '^## ' $(MAKEFILE_LIST) | sed 's/^## /  /' | column -t -s ':'
