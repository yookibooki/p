# Default binary name
BINARY ?= p
# Package path for Go commands
PKG ?= ./...
# Go command
GO ?= go
# Installation directory (override with `make INSTALL_DIR=/custom/path`)
INSTALL_DIR ?= $(HOME)/.local/bin


# Git version for build
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "unknown")
# Linker flags with version
LDFLAGS ?= -ldflags "-X main.Version=$(VERSION)"
# Output directory for binaries
BIN_DIR ?= bin

.PHONY: all fmt vet lint test tidy build release install clean check check-tools help

# Default target
all: build

## Display this help message
help:
	@echo "Makefile for building and managing the $(BINARY) Go project"
	@echo ""
	@echo "Variables (can be overridden with 'make VAR=value'):"
	@echo "  BINARY            Binary name (default: $(BINARY))"
	@echo "  PKG               Package path (default: $(PKG))"
	@echo "  GO                Go command (default: $(GO))"
	@echo "  INSTALL_DIR       Installation directory (default: $(INSTALL_DIR))"
	@echo "  VERSION           Build version (default: $(VERSION))"
	@echo "  BIN_DIR           Output directory for binaries (default: $(BIN_DIR))"
	@echo ""
	@echo "Targets:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) \
		| sort \
		| awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

## Verify required tools are installed
check-tools:
	@command -v $(GO) >/dev/null || { echo "Error: Go is not installed"; exit 1; }
	@command -v staticcheck >/dev/null || { echo "Error: staticcheck is not installed (install with 'go install honnef.co/go/tools/cmd/staticcheck@latest')"; exit 1; }
	@command -v git >/dev/null || { echo "Warning: git is not installed; version will be set to 'unknown'"; }

## Format Go code
fmt: check-tools ## Run go fmt on all packages
	@$(GO) fmt $(PKG) || { echo "Error: go fmt failed"; exit 1; }

## Run go vet
vet: check-tools ## Run go vet on all packages
	@$(GO) vet $(PKG) || { echo "Error: go vet failed"; exit 1; }

## Run static analysis
lint: check-tools ## Run staticcheck linter
	@staticcheck $(PKG) || { echo "Error: staticcheck failed"; exit 1; }

## Run tests with coverage
test: check-tools ## Run tests with coverage report
	@$(GO) test -v -parallel 4 -coverprofile=coverage.out $(PKG) || { echo "Error: tests failed"; exit 1; }
	@$(GO) tool cover -func=coverage.out

## Ensure go.mod and go.sum are tidy
tidy: check-tools ## Run go mod tidy and verify
	@$(GO) mod tidy || { echo "Error: go mod tidy failed"; exit 1; }
	@$(GO) mod verify || { echo "Error: go mod verify failed"; exit 1; }

## Build binary into $(BIN_DIR)
build: check-tools ## Build binary with version info
	@mkdir -p $(BIN_DIR)
	@$(GO) build $(LDFLAGS) -o $(BIN_DIR)/$(BINARY) $(PKG) || { echo "Error: build failed"; exit 1; }

## Run full build pipeline: fmt, vet, lint, test, tidy, build
release: check-tools fmt vet lint test tidy build ## Full safe build

## Run code quality checks (for CI)
check: check-tools fmt vet lint test tidy ## Run all checks without building

## Install binary to $(INSTALL_DIR)
install: check-tools release ## Install binary to $(INSTALL_DIR)
	@mkdir -p $(INSTALL_DIR)
	@cp -f $(BIN_DIR)/$(BINARY) $(INSTALL_DIR)/ || { echo "Error: installation failed"; exit 1; }
	@echo "Installed $(BINARY) to $(INSTALL_DIR)"





## Remove build artifacts
clean: ## Remove build artifacts
	@rm -rf $(BIN_DIR) *.out
	@echo "Cleaned build artifacts"

## Vendor dependencies
vendor: ## Run go mod vendor
	@$(GO) mod vendor || { echo "Error: go mod vendor failed"; exit 1; }
