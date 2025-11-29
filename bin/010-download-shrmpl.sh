#!/bin/bash

# Script to download shrmpl executables for local development

set -e

echo "Downloading shrmpl executables..."

# Get the directory of this script
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"

# Change to the project root directory
pushd "$PROJECT_ROOT" > /dev/null

# Create bin/tools directory if it doesn't exist
mkdir -p bin/tools

# Base URL for downloads
BASE_URL="https://github.com/wbmartin/shrmpl-srv-preview/raw/main/dist/mac"

# List of executables to download
EXECUTABLES=(
    "shrmpl-kv-srv"
    "shrmpl-log-srv"
    "shrmpl-kv-cli"
    "shrmpl-vault-srv"
)

echo "Downloading from: $BASE_URL"

for exe in "${EXECUTABLES[@]}"; do
    echo "Downloading $exe..."
    curl -L -o "bin/tools/$exe" "$BASE_URL/$exe"
    chmod +x "bin/tools/$exe"
    echo "✓ Downloaded and made executable: $exe"
done

echo ""
echo "Download complete!"
echo "Executables are in the bin/tools/ directory:"
ls -la bin/tools/


# Return to original directory
popd > /dev/null
