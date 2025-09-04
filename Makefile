BINARY      := p
INSTALL_DIR := $(HOME)/.local/bin
PKG         := ./...
GO          := go

.PHONY: all fmt vet lint test tidy build install clean

all: build

## Format code
fmt:
	$(GO) fmt $(PKG)

## Run go vet
vet:
	$(GO) vet $(PKG)

## Run static analysis (staticcheck is widely used)
lint:
	staticcheck $(PKG)

## Run tests
test:
	$(GO) test -v $(PKG)

## Ensure go.mod/go.sum are tidy
tidy:
	$(GO) mod tidy

## Build binary into ./bin
build: fmt vet test tidy
	mkdir -p bin
	$(GO) build -o bin/$(BINARY) .

## Install binary into ~/.local/bin
install: build
	mkdir -p $(INSTALL_DIR)
	cp bin/$(BINARY) $(INSTALL_DIR)/

## Remove build artifacts
clean:
	rm -rf bin
