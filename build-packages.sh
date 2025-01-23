#!/bin/bash

# Project config
PROJECT_NAME="changeIP"
VERSION="v1.0.0"
OUTPUT_DIR="builds"

# Compile configuration
GO_BUILD_FLAGS=-ldflags="-s -w"
COMPRESS_BINARIES=true
GENERATE_CHECKSUMS=true

# Platforms to compile
PLATFORMS=(
    "linux/amd64"     # Linux x86_64
    "linux/arm64"     # Linux ARM64
#    "linux/riscv64"   # Linux RISC-V
#    "linux/arm/7"     # Linux ARMv7
#    "windows/amd64"   # Windows x86_64
#    "darwin/amd64"    # macOS x86_64
#    "darwin/arm64"    # macOS ARM64
)

# Handling errors function
handle_error() {
    echo "Error: $1" >&2
    exit 1
}

# Previous requirements
command -v go >/dev/null 2>&1 || handle_error "Go is not installed"
[ -d "$OUTPUT_DIR" ] || mkdir -p "$OUTPUT_DIR" || handle_error "Can't create output directory"

# Clear previous compilations
rm -f "$OUTPUT_DIR"/* 2>/dev/null

echo "Starting compilation of $PROJECT_NAME version $VERSION..."

# Compilation for multiple platforms
for PLATFORM in "${PLATFORMS[@]}"; do
    OS=$(echo "$PLATFORM" | cut -d/ -f1)
    ARCH=$(echo "$PLATFORM" | cut -d/ -f2)
    ARM=$(echo "$PLATFORM" | cut -d/ -f3)

    # Generate the output file name
    OUTPUT="${OUTPUT_DIR}/${PROJECT_NAME}-${VERSION}-${OS}-${ARCH}"
    [ "$OS" = "windows" ] && OUTPUT+=".exe"

    echo "Compiling for $OS/$ARCH..."

    # Configuring env variables for multiple compilation
    if [ "$ARCH" = "arm" ]; then
        env GOOS="$OS" GOARCH="$ARCH" GOARM="$ARM" go build "$GO_BUILD_FLAGS" -o "$OUTPUT"
    else
        env GOOS="$OS" GOARCH="$ARCH" go build "$GO_BUILD_FLAGS" -o "$OUTPUT"
    fi

    # Check compilation state
    if [ $? -eq 0 ]; then
        echo "Compiling: $OUTPUT"

	# Compress binaries (optional)
        if [ "$COMPRESS_BINARIES" = true ] && command -v upx >/dev/null 2>&1; then
            upx --best "$OUTPUT" >/dev/null 2>&1
            echo "Binary compress with UPX"
        fi

        # Generating checksums
        if [ "$GENERATE_CHECKSUMS" = true ]; then
            (
                sha256sum "$OUTPUT" > "${OUTPUT}.sha256"
                echo "Checksums generated"
            )
        fi
    else
        echo "Compilation for $OS/$ARCH failed" >&2
    fi
done

echo -e "\033[0;32mCompilation completed. Check the directory '$OUTPUT_DIR' for the binaries.\033[0m"
