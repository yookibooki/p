# The name of your application's binary
BINARY_NAME=p

# The directory to install the binary and completions to (respecting XDG Base Directory Specification)
INSTALL_DIR?=$(HOME)/.local/bin
COMPLETION_DIR?=$(HOME)/.local/share/bash-completion/completions

# --- Main Commands ---

# The default target, executed when you run `make`. It "fixes everything".
.PHONY: all
all: fmt tidy lint test build

# Build the application binary.
.PHONY: build
build:
	@echo "==> Building $(BINARY_NAME)..."
	go build -o $(BINARY_NAME) .

# Install the binary and shell completion to your local bin.
.PHONY: install
install: build completion
	@echo "==> Installing $(BINARY_NAME) to $(INSTALL_DIR)..."
	@mkdir -p $(INSTALL_DIR)
	@install -m 0755 $(BINARY_NAME) $(INSTALL_DIR)
	@echo "$(BINARY_NAME) installed successfully."

# Run the application directly.
.PHONY: run
run:
	go run .

# Remove build artifacts.
.PHONY: clean
clean:
	@echo "==> Cleaning up..."
	@rm -f $(BINARY_NAME)
	go clean

# --- Code Quality & Dependencies ---

# Format Go source code.
.PHONY: fmt
fmt:
	@echo "==> Formatting code..."
	go fmt ./...

# Tidy and download Go module dependencies.
.PHONY: tidy
tidy:
	@echo "==> Tidying and downloading dependencies..."
	go mod tidy
	go mod download

# Run tests.
.PHONY: test
test:
	@echo "==> Running tests..."
	go test -v ./...

# Run the linter. Requires golangci-lint: https://golangci-lint.run/usage/install/
.PHONY: lint
lint:
	@echo "==> Linting code..."
	@command -v golangci-lint >/dev/null 2>&1 || (echo "golangci-lint not found. Please install it: https://golangci-lint.run/usage/install/"; exit 1)
	golangci-lint run

# --- Helpers ---

# Generate and install bash completion.
.PHONY: completion
completion: build
	@echo "==> Installing bash completion..."
	@mkdir -p $(COMPLETION_DIR)
	./$(BINARY_NAME) completion bash > $(COMPLETION_DIR)/$(BINARY_NAME)
	@echo "Bash completion installed. Please restart your shell or source your .bashrc."

# Display help information about the available commands.
.PHONY: help
help:
	@echo "Usage: make <target>"
	@echo ""
	@echo "Targets:"
	@echo "  all          Format, tidy, lint, test, and build the application (default)."
	@echo "  build        Build the application binary in the current directory."
	@echo "  install      Install the binary and shell completion to ~/.local."
	@echo "  run          Run the application without building a binary."
	@echo "  clean        Remove the built binary and other build artifacts."
	@echo ""
	@echo "Code Quality:"
	@echo "  fmt          Format the Go source code."
	@echo "  tidy         Tidy and download Go module dependencies."
	@echo "  test         Run tests."
	@echo "  lint         Run the golangci-lint linter (requires installation)."
	@echo ""
	@echo "Helpers:"
	@echo "  completion   Generate and install bash completion script."
	@echo "  help         Show this help message."