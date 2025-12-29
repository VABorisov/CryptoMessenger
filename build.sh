#!/bin/bash

set -e

APP_NAME="CryptoMessenger"
BUILD_DIR="build"
LDFLAGS="-s -w"

mkdir -p "$BUILD_DIR"

build_macos() {
    echo "Building for macOS..."

    echo "Building for macOS Intel (amd64)..."
    GOOS=darwin GOARCH=amd64 CGO_ENABLED=1 go build -ldflags "$LDFLAGS" \
        -o "$BUILD_DIR/${APP_NAME}-macos-amd64" cmd/app/main.go

    echo "Building for macOS Apple Silicon (arm64)..."
    GOOS=darwin GOARCH=arm64 CGO_ENABLED=1 go build -ldflags "$LDFLAGS" \
        -o "$BUILD_DIR/${APP_NAME}-macos-arm64" cmd/app/main.go

    echo "macOS builds complete!"
    ls -lh "$BUILD_DIR"/${APP_NAME}-macos-*
}

build_windows() {
    echo "Building for Windows..."

    echo "Building for Windows 64-bit (amd64)..."
    GOOS=windows GOARCH=amd64 CGO_ENABLED=1 go build -ldflags "$LDFLAGS" \
        -o "$BUILD_DIR/${APP_NAME}-windows-amd64.exe" cmd/app/main.go

    echo "Windows build complete!"
    ls -lh "$BUILD_DIR"/${APP_NAME}-windows-*.exe
}

build_all() {
    build_macos
    build_windows
    echo "All builds complete!"
}

case "${1:-all}" in
    macos)
        build_macos
        ;;
    windows)
        build_windows
        ;;
    all)
        build_all
        ;;
    *)
        echo "Usage: $0 [macos|windows|all]"
        exit 1
        ;;
esac
