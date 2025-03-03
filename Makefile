# Variables
BINARY_NAME=destego
BUILD_DIR=build
CMD_DIR=cmd/
VERSION=$(shell git describe --tags --always --dirty 2>/dev/null || echo "v0.0.5")
LDFLAGS=-ldflags "-X main.version=$(VERSION)"
GOPATH=$(shell go env GOPATH)
GOBIN=$(GOPATH)/bin

# Default target
.PHONY: all
all: build

# Build the binary
.PHONY: build
build:
	@echo "Building DeSteGo $(VERSION)..."
	@mkdir -p $(BUILD_DIR)
	go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) ./$(CMD_DIR)/*.go
	@echo "Build complete: $(BUILD_DIR)/$(BINARY_NAME)"

# Clean build artifacts
.PHONY: clean
clean:
	@echo "Cleaning build artifacts..."
	@rm -rf $(BUILD_DIR)
	@go clean
	@echo "Clean complete"

# Install the binary to GOBIN
.PHONY: install
install: build
	@echo "Installing DeSteGo to $(GOBIN)..."
	@cp $(BUILD_DIR)/$(BINARY_NAME) $(GOBIN)/$(BINARY_NAME)
	@echo "Installation complete"

# Uninstall the binary from GOBIN
.PHONY: uninstall
uninstall:
	@if [ -f $(GOBIN)/$(BINARY_NAME) ]; then \
		echo "Uninstalling DeSteGo from $(GOBIN)..."; \
		rm -f $(GOBIN)/$(BINARY_NAME); \
		echo "Uninstallation complete"; \
	else \
		echo "DeSteGo is not installed in $(GOBIN)"; \
	fi

# Run tests
.PHONY: test
test:
	@echo "Running tests..."
	@go test ./...

# Run the application
.PHONY: run
run: build
	@$(BUILD_DIR)/$(BINARY_NAME)

# Show help
.PHONY: help
help:
	@echo "DeSteGo Makefile"
	@echo ""
	@echo "Usage:"
	@echo "  make              Build the DeSteGo binary"
	@echo "  make build        Build the DeSteGo binary"
	@echo "  make clean        Remove build artifacts"
	@echo "  make install      Install DeSteGo to GOBIN ($(GOBIN))"
	@echo "  make uninstall    Remove DeSteGo from GOBIN ($(GOBIN))"
	@echo "  make test         Run tests"
	@echo "  make run          Build and run DeSteGo"
	@echo "  make help         Show this help message"
	@echo ""
	@echo "Current version: $(VERSION)"
