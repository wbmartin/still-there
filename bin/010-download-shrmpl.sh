#!/bin/bash

# Script to download shrmpl executables for local development
# Run this script from the project root directory

set -e

echo "Downloading shrmpl executables..."

# Create bin directory if it doesn't exist
mkdir -p bin

# Base URL for downloads
BASE_URL="https://github.com/wbmartin/shrmpl-srv-preview/raw/main/dist/mac"

# List of executables to download
EXECUTABLES=(
    "skv-srv"
    "slog-srv"
    "skv-cli"
)

echo "Downloading from: $BASE_URL"

for exe in "${EXECUTABLES[@]}"; do
    if [ -f "bin/$exe" ]; then
        echo "✓ $exe already exists, skipping..."
    else
        echo "Downloading $exe..."
        curl -L -o "bin/$exe" "$BASE_URL/$exe"
        chmod +x "bin/$exe"
        echo "✓ Downloaded and made executable: $exe"
    fi
done

echo ""
echo "Download complete!"
echo "Executables are in the bin/ directory:"
ls -la bin/

echo ""
echo "To start the services:"
echo "  ./bin/skv-srv  # KV server on port 7171"
echo "  ./bin/slog-srv # Log server on port 7379"