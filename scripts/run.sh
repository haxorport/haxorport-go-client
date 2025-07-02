#!/bin/bash

# Script to run Haxorport Client

set -e

# Project root directory
ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

# Build first
echo "Building Haxorport Client..."
./scripts/build.sh

# Run application
echo "Running Haxorport Client..."
./bin/haxorport "$@"
