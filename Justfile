# Justfile for RainBridge

# Variables
BINARY_NAME := "rainbridge"

# Default task
default:
    @just --list

# Build the application binary
build:
    @echo "Building {{BINARY_NAME}}..."
    @go build -o ./bin/{{BINARY_NAME}} ./cmd/rainbridge

# Run the application
run:
    @go run ./cmd/rainbridge

# Run unit tests
test-unit:
    @echo "Running unit tests..."
    @go test -v ./...

# Run integration tests (requires API tokens)
test-integration:
    @echo "Running integration tests..."
    @go test -v ./...

# Run all tests
test:
    @just test-unit
    @just test-integration

# Cross-compile for different platforms
build-all:
    @echo "Building for all platforms..."
    @GOOS=linux GOARCH=amd64 go build -o ./bin/{{BINARY_NAME}}-linux-amd64 ./cmd/rainbridge
    @GOOS=windows GOARCH=amd64 go build -o ./bin/{{BINARY_NAME}}-windows-amd64.exe ./cmd/rainbridge
    @GOOS=darwin GOARCH=amd64 go build -o ./bin/{{BINARY_NAME}}-darwin-amd64 ./cmd/rainbridge

# --- Packaging (Placeholders) ---

# TODO: Implement packaging for Homebrew
package-homebrew:
    @echo "Packaging for Homebrew (TODO)"

# TODO: Implement packaging for AUR
package-aur:
    @echo "Packaging for AUR (TODO)"

# TODO: Implement packaging for Fedora
package-fedora:
    @echo "Packaging for Fedora (TODO)"

# TODO: Implement packaging for Debian/Ubuntu
package-debian:
    @echo "Packaging for Debian/Ubuntu (TODO)"

# Clean up build artifacts
clean:
    @echo "Cleaning up..."
    @rm -rf ./bin
