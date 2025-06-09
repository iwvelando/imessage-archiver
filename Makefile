# Variables
BINARY_NAME=imessage-archiver
MAIN_PATH=./cmd/imessage-archiver
BUILD_DIR=./bin
GO_FILES=$(shell find . -name "*.go" -type f)

# Default target
.PHONY: all
all: build

# Build the binary
.PHONY: build
build: $(BUILD_DIR)/$(BINARY_NAME)

# Force rebuild regardless of timestamps
.PHONY: build-force
build-force:
	@mkdir -p $(BUILD_DIR)
	go build -o $(BUILD_DIR)/$(BINARY_NAME) $(MAIN_PATH)

$(BUILD_DIR)/$(BINARY_NAME): $(GO_FILES)
	@mkdir -p $(BUILD_DIR)
	go build -o $(BUILD_DIR)/$(BINARY_NAME) $(MAIN_PATH)

# Clean build artifacts
.PHONY: clean
clean:
	rm -rf $(BUILD_DIR)
	go clean

# Run tests
.PHONY: test
test:
	go test ./...

# Generate test database
.PHONY: generate-test-db
generate-test-db:
	@echo "Generating test chat.db for unit tests..."
	@chmod +x scripts/testing/generate_test_db.sh
	@scripts/testing/generate_test_db.sh

# Clean test database
.PHONY: clean-test-db
clean-test-db:
	@echo "Cleaning test database and schema..."
	@rm -f internal/archiver/testdata/chat.db
	@rm -f schema.sql

# Run tests with coverage
.PHONY: test-coverage
test-coverage:
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

# Install the binary to GOPATH/bin
.PHONY: install
install:
	go install $(MAIN_PATH)

# Install the binary to ~/bin (for macOS automation)
.PHONY: install-local
install-local: build
	@mkdir -p ~/bin
	cp $(BUILD_DIR)/$(BINARY_NAME) ~/bin/$(BINARY_NAME)
	@echo "Binary installed to ~/bin/$(BINARY_NAME)"

# Install macOS automation
.PHONY: install-macos-automation
install-macos-automation: install-local
	@echo "Installing macOS automation..."
	@chmod +x scripts/macos/install_automation.sh
	@scripts/macos/install_automation.sh

# Uninstall macOS automation
.PHONY: uninstall-macos-automation
uninstall-macos-automation:
	@echo "Uninstalling macOS automation..."
	@if [ -f "$$HOME/Library/LaunchAgents/com.imessagearchiver.plist" ]; then \
		launchctl unload "$$HOME/Library/LaunchAgents/com.imessagearchiver.plist" 2>/dev/null || true; \
		rm "$$HOME/Library/LaunchAgents/com.imessagearchiver.plist"; \
		echo "Launch agent removed"; \
	else \
		echo "Launch agent not found"; \
	fi

# Run the application with default config
.PHONY: run
run: build
	$(BUILD_DIR)/$(BINARY_NAME)

# Run the application with custom config
.PHONY: run-config
run-config: build
	$(BUILD_DIR)/$(BINARY_NAME) -config $(CONFIG)

# Format Go code
.PHONY: fmt
fmt:
	go fmt ./...

# Lint Go code (requires golangci-lint)
.PHONY: lint
lint:
	golangci-lint run

# Tidy Go modules
.PHONY: tidy
tidy:
	go mod tidy

# Build for multiple platforms
.PHONY: build-all
build-all: clean
	@mkdir -p $(BUILD_DIR)
	GOOS=darwin GOARCH=amd64 go build -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64 $(MAIN_PATH)
	GOOS=darwin GOARCH=arm64 go build -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64 $(MAIN_PATH)
	GOOS=linux GOARCH=amd64 go build -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 $(MAIN_PATH)
	GOOS=windows GOARCH=amd64 go build -o $(BUILD_DIR)/$(BINARY_NAME)-windows-amd64.exe $(MAIN_PATH)

# Help target
.PHONY: help
help:
	@echo "Available targets:"
	@echo "  build                          - Build the binary"
	@echo "  build-force                    - Force rebuild regardless of timestamps"
	@echo "  clean                          - Clean build artifacts"
	@echo "  test                           - Run tests"
	@echo "  generate-test-db               - Generate test database for unit tests"
	@echo "  clean-test-db                  - Clean test database and schema"
	@echo "  test-coverage                  - Run tests with coverage report"
	@echo "  install                        - Install binary to GOPATH/bin"
	@echo "  install-local                  - Install binary to ~/bin (for macOS automation)"
	@echo "  install-macos-automation       - Install macOS automation"
	@echo "  uninstall-macos-automation     - Uninstall macOS automation"
	@echo "  run                            - Build and run with default config"
	@echo "  run-config                     - Build and run with custom config (use CONFIG=path)"
	@echo "  fmt                            - Format Go code"
	@echo "  lint                           - Lint Go code (requires golangci-lint)"
	@echo "  tidy                           - Tidy Go modules"
	@echo "  build-all                      - Build for multiple platforms"
	@echo "  help                           - Show this help message"
