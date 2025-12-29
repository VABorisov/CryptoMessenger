.PHONY: build-macos build-windows build-all clean help


APP_NAME := CryptoMessenger
VERSION := 1.0.0
BUILD_DIR := build

LDFLAGS := -s -w
BUILD_FLAGS := -ldflags "$(LDFLAGS)"

help:
	@echo "Available targets:"
	@echo "  build-macos     - Build for macOS (Intel and Apple Silicon)"
	@echo "  build-windows  - Build for Windows (64-bit)"
	@echo "  build-all      - Build for all platforms"
	@echo "  clean          - Clean build directory"

build-macos:
	@echo "Building for macOS..."
	@mkdir -p $(BUILD_DIR)
	@echo "Building for macOS Intel (amd64)..."
	@GOOS=darwin GOARCH=amd64 CGO_ENABLED=1 go build $(BUILD_FLAGS) -o $(BUILD_DIR)/$(APP_NAME)-macos-amd64 cmd/app/main.go
	@echo "Building for macOS Apple Silicon (arm64)..."
	@GOOS=darwin GOARCH=arm64 CGO_ENABLED=1 go build $(BUILD_FLAGS) -o $(BUILD_DIR)/$(APP_NAME)-macos-arm64 cmd/app/main.go
	@echo "Build complete! Executables are in $(BUILD_DIR)/"
	@ls -lh $(BUILD_DIR)/$(APP_NAME)-macos-*

build-windows:
	@echo "Building for Windows..."
	@mkdir -p $(BUILD_DIR)
	@GOOS=windows GOARCH=amd64 CGO_ENABLED=1 go build $(BUILD_FLAGS) -o $(BUILD_DIR)/$(APP_NAME)-windows-amd64.exe cmd/app/main.go
	@echo "Build complete! Executable is in $(BUILD_DIR)/"
	@ls -lh $(BUILD_DIR)/$(APP_NAME)-windows-*.exe

build-all: build-macos build-windows
	@echo "All builds complete!"

clean:
	@echo "Cleaning build directory..."
	@rm -rf $(BUILD_DIR)
	@echo "Clean complete!"
