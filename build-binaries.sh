#!/bin/bash
# build-binaries.sh
# Script to build Guardian binaries for multiple platforms and architectures
# This creates binaries suitable for distribution via NPM

set -e  # Exit on error
VERSION="0.1.0"  # Sync this with your package.json version
BINARY_NAME="guardian"
MAIN_GO="cmd/guardian/main.go"
OUTPUT_DIR="dist/bin"  # Where to store the compiled binaries

echo "Building Guardian binaries v${VERSION}..."

# Create output directories if they don't exist
mkdir -p "${OUTPUT_DIR}"

# Clean previous builds
echo "Cleaning previous builds..."
rm -rf "${OUTPUT_DIR}"/*

# Build for each platform/architecture combination
echo "Building for multiple platforms..."

# macOS (Intel)
echo "Building for macOS (Intel)..."
GOOS=darwin GOARCH=amd64 CGO_ENABLED=0 go build -ldflags="-s -w" -o "${OUTPUT_DIR}/darwin-x64/${BINARY_NAME}" ${MAIN_GO}

# macOS (Apple Silicon)
echo "Building for macOS (Apple Silicon)..."
GOOS=darwin GOARCH=arm64 CGO_ENABLED=0 go build -ldflags="-s -w" -o "${OUTPUT_DIR}/darwin-arm64/${BINARY_NAME}" ${MAIN_GO}

# Linux (amd64)
echo "Building for Linux (amd64)..."
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags="-s -w" -o "${OUTPUT_DIR}/linux-x64/${BINARY_NAME}" ${MAIN_GO}

# Linux (arm64)
echo "Building for Linux (arm64)..."
GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -ldflags="-s -w" -o "${OUTPUT_DIR}/linux-arm64/${BINARY_NAME}" ${MAIN_GO}

# Windows (amd64)
echo "Building for Windows (amd64)..."
GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -ldflags="-s -w" -o "${OUTPUT_DIR}/win32-x64/${BINARY_NAME}.exe" ${MAIN_GO}

# Windows (arm64)
echo "Building for Windows (arm64)..."
GOOS=windows GOARCH=arm64 CGO_ENABLED=0 go build -ldflags="-s -w" -o "${OUTPUT_DIR}/win32-arm64/${BINARY_NAME}.exe" ${MAIN_GO}

# Copy built binaries to Python distribution directory
echo "Copying built binaries to dist_python/bin..."
PY_OUTPUT_DIR="dist_python/bin"
mkdir -p "${PY_OUTPUT_DIR}"
rm -rf "${PY_OUTPUT_DIR}"/*
cp -R "${OUTPUT_DIR}/." "${PY_OUTPUT_DIR}/"

# Create archives for GitHub release
echo "Creating archives for GitHub release..."
mkdir -p release

# macOS (Intel)
tar -czf "release/${BINARY_NAME}-darwin-amd64-v${VERSION}.tar.gz" -C "${OUTPUT_DIR}/darwin-x64" "${BINARY_NAME}"

# macOS (Apple Silicon)
tar -czf "release/${BINARY_NAME}-darwin-arm64-v${VERSION}.tar.gz" -C "${OUTPUT_DIR}/darwin-arm64" "${BINARY_NAME}"

# Linux (amd64)
tar -czf "release/${BINARY_NAME}-linux-amd64-v${VERSION}.tar.gz" -C "${OUTPUT_DIR}/linux-x64" "${BINARY_NAME}"

# Linux (arm64)
tar -czf "release/${BINARY_NAME}-linux-arm64-v${VERSION}.tar.gz" -C "${OUTPUT_DIR}/linux-arm64" "${BINARY_NAME}"

# Windows (amd64)
(cd "${OUTPUT_DIR}/win32-x64" && zip -q "../../../release/${BINARY_NAME}-windows-amd64-v${VERSION}.zip" "${BINARY_NAME}.exe")

# Windows (arm64)
(cd "${OUTPUT_DIR}/win32-arm64" && zip -q "../../../release/${BINARY_NAME}-windows-arm64-v${VERSION}.zip" "${BINARY_NAME}.exe")

echo "Build complete! Binaries are in ${OUTPUT_DIR}/"
echo "Archives for GitHub release are in release/"
echo "Binary sizes:"
du -h ${OUTPUT_DIR}/**/${BINARY_NAME}*